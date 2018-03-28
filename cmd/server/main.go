package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"pulse/pulser"
	"pulse/pulser/mtproto"

	"github.com/davecgh/go-spew/spew"
	"github.com/rs/cors"

	"github.com/julienschmidt/httprouter"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	forever := make(chan bool)
	// go tcpLsten()
	httpListen()
	<-forever
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	log.Println(w)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		return
	}

	conCacheData := &mtproto.CacheData{}

	data, err := mtproto.ReadHttpData(body, conCacheData)
	if err != nil {
		log.Println(err.Error())
	}

	spew.Dump(body)
	spew.Dump(data)

	pack := pulser.HandlerHttpReqPQ(data, conCacheData)
	w.Write(pack)
}

func httpListen() {
	router := httprouter.New()
	router.GET("/apiw1", Index)
	router.Handle("POST", "/apiw1", Index)

	handler := cors.Default().Handler(router)

	log.Fatal(http.ListenAndServe(":8889", handler))
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
