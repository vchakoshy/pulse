package pulser

import (
	"fmt"
	"log"
	"net"
	"pulse/pulser/mtproto"

	"github.com/davecgh/go-spew/spew"
)

type cacheData struct {
	Nonce       []byte
	ServerNonce []byte
}

func HandleConnection(conn net.Conn) error {
	fmt.Println("Handling new connection...")

	// Close connection when this function ends
	defer func() {
		fmt.Println("Closing connection...")
		conn.Close()
	}()

	conCacheData := &cacheData{}

	for {
		data, err := mtproto.ReadData(conn)

		if err != nil {
			panic(err)
		}

		switch data.(type) {
		case mtproto.TL_req_pq:
			handlerReqPQ(data, conn, conCacheData)
		case mtproto.TL_req_DH_params:
			handlerReqDHParams(data, conn, conCacheData)

		default:
			spew.Dump(data)
			log.Println("handler not found")
		}

	}
}
