package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/vchakoshy/pulse/pulser"
	"github.com/vchakoshy/pulse/pulser/mtproto"
)

var (
	sessions = make(map[string]*mtproto.CacheData)
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	r := pulser.GetRsaKey()
	log.Println(r.PrivateKey.E)
	forever := make(chan bool)
	// go tcpLsten()
	httpListen()
	<-forever
}

func httpListen() {
	http.HandleFunc("/apiw1", pulser.HandleHttpConnection)
	http.ListenAndServe(":8889", nil)
}

func tcpLsten() {
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
