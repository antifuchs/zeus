package main

import (
	"github.com/burke/zeus/go/filemonitor"
	"io"
	"net"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		println("USAGE: " + os.Args[0] + " listen-port")
		os.Exit(1)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:"+os.Args[1])
	if err != nil {
		println("Couldn't start listening on " + os.Args[1] + ":" + err.Error())
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			println("Error accepting a connection: " + err.Error())
			os.Exit(1)
		}
		go handleConnection(conn)
	}

}

func handleConnection(conn net.Conn) {
	cmd, watcherIn, watcherOut, _, err := filemonitor.StartWrapperProgram()
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	if err != nil {
		println("Couldn't start the host-side filemonitor wrapper: " + err.Error())
		os.Exit(1)
	}

	go io.Copy(conn, watcherOut)
	io.Copy(watcherIn, conn)
}
