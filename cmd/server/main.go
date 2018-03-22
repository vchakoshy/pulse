package main

import (
	"fmt"
	"log"
	"net"
	"pulse/pulser"
)

func main() {
	log.Println("listen at: 127.0.0.1:8888")
	listener, err := net.Listen("tcp", ":8888")
	if err != nil {
		fmt.Println(err)
		return
	}

	defer func() {
		listener.Close()
		fmt.Println("Listener closed")
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			break
		}

		go pulser.HandleConnection(conn)
	}
}
