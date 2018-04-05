package pulser

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"pulse/pulser/mtproto"
	"time"

	"github.com/davecgh/go-spew/spew"
)

var (
	sessions      = make(map[string]*mtproto.CacheData)
	authKeyHashes = make(map[string]*mtproto.CacheData)
)

func HandleConnection(conn net.Conn) error {
	fmt.Println("Handling new connection...")

	defer func() {
		fmt.Println("Closing connection...")
		conn.Close()
	}()

	conCacheData := &mtproto.CacheData{}

	for {
		data, err := mtproto.ReadData(conn, conCacheData)

		if err != nil {
			panic(err)
		}

		switch data.(type) {
		case mtproto.TL_req_pq:
			go handlerReqPQ(data, conn, conCacheData)
		case mtproto.TL_req_DH_params:
			go handlerReqDHParams(data, conn, conCacheData)
		case mtproto.TL_set_client_DH_params:
			go handlerSetClientDHParams(data, conn, conCacheData)
		case mtproto.TL_invokeWithLayer:
			go handlerinvokeWithLayer(data, conn, conCacheData)
		case mtproto.TL_msgs_ack:
			go handlerMsgsAck(data, conn, conCacheData)

		default:
			spew.Dump(data)
			log.Println("handler not found")
		}

	}
}

var inSleep = false
var aliveConnection http.ResponseWriter

var c2 chan []byte

func init() {
	c2 = make(chan []byte, 1)
}

func HandleHttpConnection(w http.ResponseWriter, r *http.Request) {
	c2 = make(chan []byte, 1)
	// defer close(c2)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// log.Println(w)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		return
	}

	if len(body) == 0 {
		return
	}

	// cd := &mtproto.CacheData{}

	hData, err := mtproto.ReadHttpData(body, authKeyHashes)
	if err != nil {
		log.Println(err.Error())
		return
	}

	spew.Dump(hData.Data)
	aliveConnection = w
	go handleHdata(hData.Data, w, &hData.CacheData)

	dur := time.Duration(5000) * time.Millisecond

	timer := time.NewTimer(dur)
	defer timer.Stop()

	select {
	case res := <-c2:
		w.Write(res)

	case <-timer.C:
		w.Header().Set("im sleep", "oonghadr")
		w.Write(makeEmptyPacket(&hData.CacheData))
		log.Println("time out")

	}

}

func handleHdata(d interface{}, w http.ResponseWriter, cd *mtproto.CacheData) {
	switch d.(type) {
	case mtproto.TL_req_pq:
		pack := HandlerHttpReqPQ(d, sessions)
		c2 <- pack
		// w.Write(pack)
	case mtproto.TL_req_DH_params:
		pack := HandlerHttpReqDHParams(d, sessions)
		c2 <- pack
		// w.Write(pack)
	case mtproto.TL_set_client_DH_params:
		pack := HandlerHttpSetClientDHParams(d, sessions)
		c2 <- pack
		// w.Write(pack)
	case mtproto.TL_ping:
		// log.Println("ping recive")
	case mtproto.TL_invokeWithLayer:
		pack := handlerHttpInvokeWithLayer(d.(mtproto.TL_invokeWithLayer), cd)
		c2 <- pack
		// w.Header().Set("media", "long")
		// w.Write(pack)
	case mtproto.TL_auth_sendCode:
		pack := handlerHttpAuthSendCode(cd)
		c2 <- pack
	case mtproto.TL_msg_container:
		msgContainer := d.(mtproto.TL_msg_container)

		for _, item := range msgContainer.Items {
			cd.ReqMsgID = item.Msg_id
			handleHdata(item.Data, w, cd)
		}

	case mtproto.TL_http_wait:

		log.Println("mtproto.TL_http_wait recieved")
		// aliveConnection = w
		// httpWait := d.(mtproto.TL_http_wait)

	default:
		spew.Dump(d)
		log.Println("no handler found")
	}
}

func makeEmptyPacket(cd *mtproto.CacheData) []byte {
	resultContainer := mtproto.TL_msg_container{}

	cd.SeqNo++
	newSessionCreated := mtproto.TL_MT_message{
		Msg_id: mtproto.GenerateMessageId(),
		Seq_no: cd.SeqNo,
		Size:   28,
		Data:   mtproto.TL_msgs_ack{},
	}
	resultContainer.Items = append(resultContainer.Items, newSessionCreated)
	pack, err := mtproto.MakePacketHttp(resultContainer, cd)
	if err != nil {
		panic(err)
	}
	return pack
}

// var sendChannel = make(chan []mtproto.TL_MT_message)

// func init() {
// 	sendChannel:= make(chan []mtproto.TL_MT_message,0)
// }

// func sender(){

// }
