package main

import (
	"fmt"
	"net"
	"os"
)

func main() {

	var (
		TCP_CONN_PORT = "115"
	)

	l, err := net.Listen("tcp", "localhost"+":"+TCP_CONN_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}

	defer l.Close()
	fmt.Println("Listening on " + "localhost" + ":" + TCP_CONN_PORT)
	for {

		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {

	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Empty message")
	}

	conn.Write([]byte("Message received."))
	conn.Close()
}
