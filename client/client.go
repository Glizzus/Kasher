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
	localAddr   string
	serverHost  string
	destination string
}

func parseArgs() *kasherArgs {
	args := os.Args[1:]
	kasherArgs := kasherArgs{
		localAddr:   args[0],
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

var buff = make([]byte, 1024*640)

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
				if res.StatusCode == http.StatusNoContent {
					log.Println("Nothing")
					continue
				}
				fmt.Println("Bad http response while polling ", res.StatusCode)
				connected = false
			}
			io.Copy(conn, res.Body)
		}
	}()
	for connected {
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		n, err := conn.Read(buff)
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
		if n == 0 {
			continue
		}
		send, err := http.NewRequest(http.MethodPut, hostUrl, bytes.NewBuffer(buff[:n]))
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
	log.Printf("Opening local address %s", args.localAddr)
	addr, err := net.ResolveTCPAddr("tcp", args.localAddr)
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
