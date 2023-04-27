package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
	"unicode"
)

type kasherClient struct {
	conn          *net.TCPConn
	buffer        []byte
	lengthChannel chan int
	errorChannel  chan error
}

// We use a map to keep track of each connection.
// The map is indexed by a unique connection ID, which leads
// To the corresponding TCP connection
var connectionMap = make(map[string]*kasherClient)

func (client *kasherClient) readNonBlocking() (int, error) {
	err := client.conn.SetReadDeadline(time.Now().Add(time.Second * 30))
	if err != nil {
		log.Println("Error setting read deadline: ", err.Error())
	}
	go func() {
		length, err := client.conn.Read(client.buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				client.lengthChannel <- 0
			} else {
				client.errorChannel <- err
			}
		} else {
			client.lengthChannel <- length
		}
	}()

	select {
	case length := <-client.lengthChannel:
		return length, nil
	case err := <-client.errorChannel:
		return 0, err
	}
}

// Deletes a connection by removing the connection entry
// from our connection map.
func doDelete(connId string, w *http.ResponseWriter) {
	delete(connectionMap, connId)
	log.Printf("Connection deleted for connection %s", connId)
	(*w).WriteHeader(http.StatusOK)
}

// Our primary connection handler. It takes responses in this format:
//
//	/[connId]
//
// And performs operations depending upon the method used.
//
// DELETE: Removes connection with given id
//
// POST: Creates connection with given id
//
// PUT: Takes data from request and ferries it to the destination
//
// GET: Sends data from the destination to the requester
func connectionHandler(w http.ResponseWriter, r *http.Request) {
	connId := r.URL.Path[1:]
	switch r.Method {

	case http.MethodDelete:
		doDelete(connId, &w)

	case http.MethodPost:
		log.Printf("New connection %s from %s", connId, r.RemoteAddr)
		destBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println("Error reading body from POST: ", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
		destination := string(destBytes)
		remoteAddr, err := net.ResolveTCPAddr("tcp", destination)
		if err != nil {
			log.Println(err.Error())
		}
		log.Printf("Attempting to open connection")
		conn, err := net.DialTCP("tcp", nil, remoteAddr)
		if err != nil {
			log.Println("Error dialing destination: ", err.Error())
		}
		_ = conn.SetKeepAlive(true)

		client := kasherClient{
			conn:          conn,
			buffer:        make([]byte, 1024),
			lengthChannel: make(chan int),
			errorChannel:  make(chan error),
		}
		connectionMap[connId] = &client
		w.WriteHeader(http.StatusCreated)
	case http.MethodPut:
		client, exists := connectionMap[connId]
		if !exists {
			log.Println("Received PUT for nonexistent connection ", connId)
			w.WriteHeader(http.StatusNotFound)
		}
		_, err := io.Copy(client.conn, r.Body)
		if err != nil {
			log.Println("Error copying body to connection", err.Error())
		}
	case http.MethodGet:
		client, exists := connectionMap[connId]
		if !exists {
			log.Println("Received GET for nonexistent connection ", connId)
			w.WriteHeader(http.StatusNotFound)
		}
		length, err := client.readNonBlocking()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				fmt.Println("Connection closed from remote host, removing connection")
				w.WriteHeader(http.StatusGone)
				return
			}
			fmt.Println("Error on read from local socket: ", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if length == 0 {
			log.Println("Received GET but no information from socket, continuing...")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Write(client.buffer[:length])
		return
	}
}

func main() {

	localPort := os.Args[1]
	if localPort == "" {
		log.Fatal("Local port is undefined")
	}
	for _, c := range localPort {
		if !unicode.IsDigit(c) {
			log.Fatal("Local port must be a positive number")
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", connectionHandler)

	server := &http.Server{
		Addr:      "0.0.0.0:" + localPort,
		TLSConfig: SelfSignedCertificate(),
		Handler:   mux,
	}
	log.Printf("Attempting to listen on port %s", localPort)
	err := server.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatal(err.Error())
	}
}
