package main

import (
	"bytes"
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

func generateUuid() string {
	id := make([]byte, 16)
	rand.Read(id)
	return string(id)
}

type kasherArgs struct {
	localPort       int
	serverHost      *url.URL
	destinationHost string
	destinationPort int
}

func parseArgs() *kasherArgs {
	args := os.Args[1:]
	localPort, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatalf("Supplied local port %d is not a number\n", localPort)
	}
	serverHost, err := url.Parse(args[1])
	if err != nil {
		log.Fatal("Invalid server host: ", err.Error())
	}
	destinationHost := args[2]
	destinationPort, err := strconv.Atoi(args[3])
	if err != nil {
		log.Fatalf("Supplied destination port %d is not a number\n", destinationPort)
	}
	kargs := kasherArgs{
		localPort:       localPort,
		serverHost:      serverHost,
		destinationHost: destinationHost,
		destinationPort: destinationPort,
	}
	return &kargs
}

func randomMethod() string {
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	return methods[rand.Intn(3)]
}

var client = &http.Client{
	Timeout: 63 * time.Second,
}

func handleConnection(conn *net.TCPConn, args *kasherArgs) {
	connId := generateUuid()

	fmt.Printf("Opening connection for %s%d (%s)", args.destinationHost, args.destinationPort, connId)
	hostUrl := args.serverHost.String() + "/" + connId
	data := fmt.Sprintf("%s:%d", args.destinationHost, args.destinationPort)
	req, err := http.NewRequest(http.MethodPost, hostUrl, bytes.NewBufferString(data))
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
				fmt.Println("Bad http response while polling ", res.StatusCode)
				connected = false
			}

		}
	}()
	var maxBufferSize int64 = 1024 * 640
	for connected {
		limitedReader := io.LimitReader(conn, maxBufferSize)
		data, err := io.ReadAll(limitedReader)
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
		send, err := http.NewRequest(http.MethodPut, hostUrl, bytes.NewBuffer(data))
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
	log.Printf("Opening local port %d", args.localPort)
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", args.localPort))
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
