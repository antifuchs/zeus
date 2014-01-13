package main

import (
	"bufio"
	"fmt"
	"github.com/burke/zeus/go/filemonitor"
	"io"
	"net"
	"os"
	"strings"
)

var hostToGuest map[string]string = make(map[string]string)
var guestToHost map[string]string = make(map[string]string)

var verbose bool

func main() {
	verbose = true
	if len(os.Args) < 2 {
		println("USAGE: " + os.Args[0] + " listen-port [hostDir:guestDir] ...")
		os.Exit(1)
	}
	port := os.Args[1]

	dirMappings := os.Args[2:]
	for _, mapping := range dirMappings {
		dirs := strings.Split(mapping, ":")
		if len(dirs) != 2 {
			panic("Encountered malformed dir mapping spec " + mapping)
		}
		hostToGuest[dirs[0]] = dirs[1]
		guestToHost[dirs[1]] = dirs[0]
	}

	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		panic("Couldn't start listening on " + os.Args[1] + ":" + err.Error())
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic("Error accepting a connection: " + err.Error())
		}
		go handleConnection(conn)
	}

}

func handleConnection(conn net.Conn) {
	cmd, watcherIn, watcherOut, _, err := filemonitor.StartWrapperProgram()
	if err != nil {
		panic("Couldn't start the host-side filemonitor wrapper: " + err.Error())
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	readErrors := make(chan error)
	go proxyFiles(readErrors, bufio.NewWriter(conn), bufio.NewReader(watcherOut), hostToGuest, true)
	go proxyFiles(readErrors, bufio.NewWriter(watcherIn), bufio.NewReader(conn), guestToHost, false)
	select {
	case err = <-readErrors:
		if err != io.EOF {
			println("Error handling the connection: " + err.Error())
		}
	}
}

func proxyFiles(readErrors chan error, out *bufio.Writer, in *bufio.Reader, translation map[string]string, eofIsFatal bool) {
	for {
		originalFile, err := in.ReadString('\n')
		if err != nil {
			readErrors <- err
			return
		}
		file := originalFile

		// TODO: I'm sure this is plenty inefficient:
		for dir, replacement := range translation {
			if strings.HasPrefix(file, dir) {
				file = replacement + file[len(dir):]
			}
		}
		if verbose {
			fmt.Printf("Got a file notification: %V => %V", originalFile, file)
		}
		out.WriteString(file)
		out.Flush()
	}
}
