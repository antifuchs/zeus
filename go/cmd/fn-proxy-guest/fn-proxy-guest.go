package main

import (
	"github.com/burke/zeus/go/filemonitor"
	"net"
	"os"

	"io"
)

func main() {
	port := os.Getenv(filemonitor.PROXY_PORT_ENV_VAR)

	if port == "" {
		println("USAGE: " + filemonitor.PROXY_PORT_ENV_VAR + "=port_number " + os.Args[0])
		os.Exit(1)
	}

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	defer conn.Close()
	if err != nil {
		println("Could not connect to socket " + port + ": " + err.Error())
		os.Exit(1)
	}
	go io.Copy(conn, os.Stdin)
	io.Copy(os.Stdout, conn)
}
