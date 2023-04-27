package main

import (
	"bytes"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

func main() {
	localPort, serverHost, destination := parseArgs()
	log.Printf("Opening local port %s", localPort)

	listener := createListener(localPort)
	runListenerForever(listener, serverHost, destination)
	if err := listener.Close(); err != nil {
		log.Println("Error closing TcpListener: ", err.Error())
	}
}

func createListener(port string) *net.TCPListener {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:"+port)
	if err != nil {
		log.Fatal(err.Error())
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal("Unable to start TCP listener: ", err.Error())
	}
	return listener
}

func runListenerForever(listener *net.TCPListener, serverHost, destination string) {
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Println("Error accepting incoming connection: ", err.Error())
		}
		if err := conn.SetKeepAlive(true); err != nil {
			log.Println("Unable to set keepalive: ", err.Error())
		}
		go handleConnection(conn, serverHost, destination)
	}
}

// The HTTP Client that we will be using to make requests
var client = &http.Client{
	Timeout: 63 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

type kasherHttpRequester = func(string, io.Reader) (*http.Request, error)

// Creates a closure that allows http.Requests to be created with base URLs
func newKasherHttpRequest(baseUrl string) kasherHttpRequester {
	return func(method string, body io.Reader) (*http.Request, error) {
		return http.NewRequest(method, baseUrl, body)
	}
}

// Creates the connection and runs the GET and PUT loops, while cleaning up if necessary
func handleConnection(conn *net.TCPConn, serverHost string, destination string) {
	connId := generateUuid()
	log.Printf("Opening connection for %s (%s)\n", destination, connId)
	hostUrl := serverHost + "/" + connId
	kasherRequester := newKasherHttpRequest(hostUrl)

	if err := postConnection(kasherRequester, destination); err != nil {
		return
	}

	connected := true
	go getLoop(&connected, hostUrl, conn)
	putLoop(&connected, kasherRequester, conn)
}

func postConnection(kasherRequest kasherHttpRequester, destination string) error {
	req, err := kasherRequest(http.MethodPost, bytes.NewBufferString(destination))
	if err != nil {
		log.Println("Error creating http request: ", err.Error())
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		log.Println("Error with http response: ", err.Error())
		return err
	}
	if res.StatusCode != http.StatusCreated {
		switch res.StatusCode {
		case http.StatusInternalServerError:
			log.Println("Unknown server error trying to establish connection")
		}
		return fmt.Errorf("bad status received while posting connection: %d", res.StatusCode)
	}
	return nil
}

func getLoop(connected *bool, hostUrl string, conn *net.TCPConn) {
	// If we fail too much, give up
	failCount := 0
	maxFails := 5
	for *connected {
		res, err := client.Get(hostUrl)
		if err != nil {
			if failCount < maxFails {
				failCount++
				continue
			}
			log.Printf("GET has failed %d times, ending connection", maxFails)
			*connected = false
		}
		failCount = 0
		if res.StatusCode != http.StatusOK {
			switch res.StatusCode {
			case http.StatusNoContent:
				log.Println("Received nothing but OK, continuing...")
				continue
			case http.StatusGone:
				log.Println("Socket connection closed, returning...")
				*connected = false
			default:
				log.Println("Bad http response while polling ", res.StatusCode)
				*connected = false
			}
		} else {
			if _, err := io.Copy(conn, res.Body); err != nil {
				log.Println("Error sending response body to local socket: ", err.Error())
			}
		}
	}
}

func putLoop(connected *bool, requester kasherHttpRequester, conn *net.TCPConn) {
	lengthChannel := make(chan int)
	errorChannel := make(chan error)
	buffer := make([]byte, 1024*640)

	failCount := 0
	maxFails := 5
	for *connected {
		length, err := readNonBlocking(conn, buffer, lengthChannel, errorChannel)
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Println("Local connection closed")
				log.Println("Closing connection to remote location")
				del, err := requester(http.MethodDelete, nil)
				if err != nil {
					log.Println("Error creating request from remote: ", err.Error())
				}
				_, err = client.Do(del)
				if err != nil {
					log.Println(err.Error())
				}
				*connected = false
			} else {
				if failCount < maxFails {
					failCount++
					continue
				}
				log.Printf("Failed PUT %d times, exiting", maxFails)
				*connected = false
			}
		}
		failCount = 0
		// If we read nothing, don't send it
		if length == 0 {
			continue
		}
		send, err := requester(http.MethodPut, bytes.NewBuffer(buffer[:length]))
		if err != nil {
			log.Println(err.Error())
		}
		if _, err = client.Do(send); err != nil {
			log.Println(err.Error())
		}
	}
}

// Generates a random length 16 hex string
func generateUuid() string {
	id := make([]byte, 16)
	rand.Read(id)
	return hex.EncodeToString(id)
}

func parseArgs() (localPort string, serverHost string, destination string) {
	switch length := len(os.Args); {
	case length < 4:
		panic("Not enough arguments provided, 3 expected")
	case length > 4:
		panic("Too many arguments provided, 3 expected")
	}
	localPort = os.Args[1]
	if _, err := strconv.ParseUint(localPort, 10, 16); err != nil {
		log.Panicf("Invalid port: %s", localPort)
	}
	serverHost = os.Args[2]
	if _, err := url.Parse(serverHost); err != nil {
		log.Panicf("Error while parsing server host url: %s", err.Error())
	}
	destination = os.Args[3]
	return localPort, serverHost, destination
}

func readNonBlocking(conn *net.TCPConn, buffer []byte, lengthChannel chan int, errorChannel chan error) (int, error) {
	err := conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		log.Println("Error setting read deadline on connection: ", err.Error())
	}
	go func() {
		n, err := conn.Read(buffer)
		if err == nil || errors.Is(err, os.ErrDeadlineExceeded) {
			lengthChannel <- n
		} else {
			errorChannel <- err
		}
	}()

	select {
	case err := <-errorChannel:
		return 0, err
	case n := <-lengthChannel:
		return n, nil
	}
}
