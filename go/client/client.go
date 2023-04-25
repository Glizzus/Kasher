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
	"os"
	"time"
)

func generateUuid() string {
	id := make([]byte, 16)
	rand.Read(id)
	return hex.EncodeToString(id)
}

type kasherArgs struct {
	localPort   string
	serverHost  string
	destination string
}

func parseArgs() *kasherArgs {
	args := os.Args[1:]
	kasherArgs := kasherArgs{
		localPort:   args[0],
		serverHost:  args[1],
		destination: args[2],
	}
	return &kasherArgs
}

var client = &http.Client{
	Timeout: 63 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func readNonBlocking(conn *net.TCPConn, buffer []byte, lengthChannel chan int, errorChannel chan error) (int, error) {
	go func() {
		length, err := conn.Read(buffer)
		if err != nil {
			errorChannel <- err
		} else {
			lengthChannel <- length
		}
	}()

	select {
	case length := <-lengthChannel:
		return length, nil
	case err := <-errorChannel:
		return 0, err
	}
}

func handleConnection(conn *net.TCPConn, args *kasherArgs) {
	connId := generateUuid()

	log.Printf("Opening connection for %s (%s)\n", args.destination, connId)
	hostUrl := args.serverHost + "/" + connId
	req, err := http.NewRequest(http.MethodPost, hostUrl, bytes.NewBufferString(args.destination))
	if err != nil {
		fmt.Println("Error creating http request: ", err.Error())
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error with http response: ", err.Error())
		return
	}
	// TODO: Implement switching on error codes to provide more informative error messages
	if res.StatusCode != http.StatusCreated {
		fmt.Printf("Bad http response %d received", res.StatusCode)
		switch res.StatusCode {
		case http.StatusInternalServerError:
			fmt.Println("Unknown server error")
		}
		return
	}
	connected := true
	go func() {
		for connected {
			res, err := client.Get(hostUrl)
			if err != nil {
				fmt.Println("Polling failed")
				connected = false
			}
			// TODO: Implement switching on error codes to provide more informative error messages
			if res.StatusCode != http.StatusOK {
				switch res.StatusCode {
				case http.StatusNoContent:
					log.Println("Received nothing but OK, continuing...")
					continue
				case http.StatusGone:
					log.Println("Socket connection closed, returning...")
					connected = false
				default:
					log.Println("Bad http response while polling ", res.StatusCode)
					connected = false
				}
			} else {
				io.Copy(conn, res.Body)
			}
		}
	}()
	lengthChannel := make(chan int)
	errorChannel := make(chan error)
	buffer := make([]byte, 1024 * 640)
	for connected {
		length, err := readNonBlocking(conn, buffer, lengthChannel, errorChannel)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("Local connection closed")
				fmt.Println("Closing connection to remote location")
				del, err := http.NewRequest(http.MethodDelete, hostUrl, nil)
				if err != nil {
					fmt.Println(err.Error())
				}
				_, err = client.Do(del)
				if err != nil {
					fmt.Println(err.Error())
				}
				connected = false
			}
		}
		if length == 0 {
			continue
		}
		send, err := http.NewRequest(http.MethodPut, hostUrl, bytes.NewBuffer(buffer[:length]))
		if err != nil {
			fmt.Println(err.Error())
		}
		res, err = client.Do(send)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func main() {

	args := parseArgs()
	log.Printf("Opening local port %s", args.localPort)
	addr, err := net.ResolveTCPAddr("tcp", "localhost:"+args.localPort)
	if err != nil {
		log.Fatal(err.Error())
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal("Unable to start TCP listener: ", err.Error())
	}
	defer func() {
		err := listener.Close()
		if err != nil {
			log.Fatal(err.Error())
		}
	}()
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Println("Error accepting incoming connection: ", err.Error())
		}
		err = conn.SetKeepAlive(true)
		if err != nil {
			log.Println("Unable to set keepalive", err.Error())
		}
		go handleConnection(conn, args)
	}
}
