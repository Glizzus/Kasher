package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

var conn, _ = net.Dial("tcp", ":22")

var stringExchanged = false
var algsExchanged = false

var reader = bufio.NewReader(conn)

func sendExchange(w http.ResponseWriter, conn net.Conn) {
	buffer := make([]byte, 2048)
	len, err := io.ReadAtLeast(conn, buffer, 512)
	if err != nil {
		log.Println(err.Error())
	}
	_, err = w.Write(buffer[:len])
	if err != nil {
		log.Println(err.Error())
	}
}

func serviceUnavailable(res http.ResponseWriter, message string) {
	http.Error(res, message, http.StatusInternalServerError)
}

func dialDestination(res http.ResponseWriter, req *http.Request) (net.Conn, error) {

	destConn, err := net.DialTimeout("tcp", req.Host, 10*time.Second)
	if err != nil {
		serviceUnavailable(res, err.Error())
		return nil, err
	}
	return destConn, nil
}

func transfer(dst io.WriteCloser, src io.ReadCloser) {

	close := func(closer io.Closer) {
		err := closer.Close()
		if err != nil {
			log.Println(err.Error())
		}
	}
	defer close(dst)
	defer close(src)
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Println(err.Error())
	}
}

func getBody(res http.ResponseWriter, req *http.Request) {
	destConn, err := dialDestination(res, req)
	if err != nil {
		return
	}
	res.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(res, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		serviceUnavailable(res, err.Error())
	}
	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)

}

func main() {
	server := &http.Server{
		Addr: ":10000",
		Handler: http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		}),
	}
	http.HandleFunc("/", getBody)
	port := ":10000"
	log.Println("Serving...")
	err := http.ListenAndServeTLS(port, "./cert.pem", "./key.pem", nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
