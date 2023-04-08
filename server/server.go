package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

var connectionMap = make(map[string]*net.TCPConn)

func connectionHandler(w http.ResponseWriter, r *http.Request) {
	var maxBufferSize int64 = 1024 * 640
	connId := r.URL.Path[1:]
	// TODO: Make these different functions
	switch r.Method {
	case http.MethodDelete:
		delete(connectionMap, connId)
		log.Printf("Connection deleted for connection %s", connId)
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

		connectionMap[connId] = conn
		w.WriteHeader(http.StatusCreated)
	case http.MethodPut:
		conn, exists := connectionMap[connId]
		if !exists {
			log.Println("Received PUT for nonexistent connection ", connId)
			w.WriteHeader(http.StatusNotFound)
		}
		_, err := io.Copy(conn, r.Body)
		if err != nil {
			log.Println("Error copying body to connection", err.Error())
		}
	case http.MethodGet:
		conn, exists := connectionMap[connId]
		if !exists {
			log.Println("Received GET for nonexistent connection ", connId)
			w.WriteHeader(http.StatusNotFound)
		}
		_ = conn.SetReadDeadline(time.Now().Add(time.Millisecond * 10))
		_, _ = io.CopyN(w, conn, maxBufferSize)
	}
}

func parseArgs() (string, string) {
	var cert string
	flag.StringVar(&cert, "cert", "", "The filepath of the certificate file")
	var key string
	flag.StringVar(&key, "key", "", "The filepath of the key file")
	flag.Parse()

	if cert == "" || key == "" {
		if cert == key {
			log.Fatal("You must define the path to your SSL certificate and key using the --cert and --key flags")
		}
		if cert == "" {
			log.Fatal("You must define the path to your SSL certificate using the --cert flag")
		}
		log.Fatal("You must define the path to your SSL key using the --key flag")
	}
	return cert, key
}

func main() {

	cert, key := parseArgs()
	http.HandleFunc("/", connectionHandler)
	err := http.ListenAndServeTLS(":10000", cert, key, nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
