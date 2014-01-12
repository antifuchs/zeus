package main

import (
	"github.com/burke/zeus/go/filemonitor"
	"net"
	"os"

	"bytes"
	"errors"
	"fmt"
	"io"
	"time"
)

const BUFSIZE = 4096

var savedRequests *bytes.Buffer

func main() {
	port := os.Getenv(filemonitor.PROXY_PORT_ENV_VAR)

	if port == "" {
		println("USAGE: " + filemonitor.PROXY_PORT_ENV_VAR + "=port_number " + os.Args[0])
		os.Exit(1)
	}

	savedRequests = bytes.NewBuffer(make([]byte, BUFSIZE))
	saveReader, saveWriter := io.Pipe()
	savingInputReader := io.TeeReader(os.Stdin, saveWriter)
	go savedRequests.ReadFrom(saveReader)

	connections := make(chan net.Conn)
	requests := make(chan *bytes.Buffer)
	go proxyRequests(connections, requests)

	for {
		if err := attemptConnection(port, savingInputReader, connections, requests); err != nil {
			// TODO: handle that error gracefully. This
			// sleep is mostly there just so we don't peg
			// the CPU forever:
			println("Connection error (" + err.Error() + ") - retrying in 1s...")
			time.Sleep(1 * time.Second)
		} else {
			return
		}
	}
}

func proxyRequests(connections chan net.Conn, requests chan *bytes.Buffer) {
	var req *bytes.Buffer

	req = nil
	conn := <-connections

	for req == nil {
		select {
		case conn = <-connections:
		case req = <-requests:
		}
	}
	for {
		fmt.Printf("Sending to %v...\n", conn)
		if _, err := req.WriteTo(conn); err != nil {
			println("Error proxying a request over: " + err.Error() + ". Dumping this connection and getting an new one.")
			conn = <- connections
			continue
		}
		select {
		case conn = <-connections:
		case req = <-requests:
		}
	}
}

func attemptConnection(port string, input io.Reader, connections chan net.Conn, requests chan *bytes.Buffer) error {
	println("Trying to connect to port " + port)

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		return err
	}
	defer conn.Close()

	if savedRequests.Len() > 0 {
		println("Connected - replaying...")
		// If we have any requests saved (e.g. if the connection
		// broke), replay it here:
		if _, err = conn.Write(savedRequests.Bytes()); err != nil {
			return err
		}
	}
	connections <- conn
	println("Replay done - now proxying requests.")

	failure := make(chan error)
	done := make(chan bool)
	go readStdin(failure, requests, done, input)
	go copyInputFromNotifier(failure, os.Stdout, conn)

	for {
		select {
		case err = <-failure:
			return err
		case <-done:
			return nil
		}
	}
}

func copyInputFromNotifier(failure chan error, to io.Writer, from io.Reader) {
	_, err := io.Copy(to, from)
	if err != nil {
		failure <- err
	}
	failure <- errors.New("Got an EOF where I shouldn't have")
}

func readStdin(failure chan error, requests chan *bytes.Buffer, done chan bool, from io.Reader) {
	buf := make([]byte, BUFSIZE)
	for {
		length, err := from.Read(buf)
		if err != nil {
			failure <- err
			return
		}
		if length == 0 {
			done <- true
			return
		}
		req := bytes.NewBuffer(make([]byte, length))
		req.Write(buf[:length])
		requests <- req
	}
}
