package mtproto

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"reflect"
	"time"

	"github.com/davecgh/go-spew/spew"
	redis "gopkg.in/redis.v4"
)

func (m *MTProto) SendPacket(msg TL, resp chan response) error {
	return m.sendPacket(msg, resp)
}

func (m *MTProto) sendPacket(msg TL, resp chan response) error {
	log.Println("send packet ", msg)
	obj := msg.encode()
	// log.Println("MTProto::sendPacket::", reflect.TypeOf(msg).String())
	x := NewEncodeBuf(256)

	// padding for tcpsize
	x.Int(0)

	if m.encrypted {
		needAck := true
		switch msg.(type) {
		case TL_ping, TL_msgs_ack:
			needAck = false
		}
		z := NewEncodeBuf(256)
		newMsgId := GenerateMessageId()
		z.Bytes(m.serverSalt)
		z.Long(m.sessionId)
		z.Long(newMsgId)
		if needAck {
			z.Int(m.lastSeqNo | 1)
		} else {
			z.Int(m.lastSeqNo)
		}
		z.Int(int32(len(obj)))
		z.Bytes(obj)

		msgKey := sha1(z.buf)[4:20]
		aesKey, aesIV := generateAES(msgKey, m.authKey, false)

		y := make([]byte, len(z.buf)+((16-(len(obj)%16))&15))
		copy(y, z.buf)
		encryptedData, err := doAES256IGEencrypt(y, aesKey, aesIV)
		if err != nil {
			return err
		}

		m.lastSeqNo += 2
		if needAck {
			m.mutex.Lock()
			m.msgsIdToAck[newMsgId] = packetToSend{msg, resp}
			m.mutex.Unlock()
		}

		x.Bytes(m.authKeyHash)
		x.Bytes(msgKey)
		x.Bytes(encryptedData)

		if resp != nil {
			m.mutex.Lock()
			m.msgsIdToResp[newMsgId] = resp
			m.mutex.Unlock()
		}

	} else {
		x.Long(0)
		x.Long(GenerateMessageId())
		x.Int(int32(len(obj)))
		x.Bytes(obj)
	}

	// minus padding
	size := len(x.buf)/4 - 1

	if size < 127 {
		x.buf[3] = byte(size)
		x.buf = x.buf[3:]
	} else {
		binary.LittleEndian.PutUint32(x.buf, uint32(size<<8|127))
	}

	n, err := m.conn.Write(x.buf)
	log.Println("write buffer to connection", n)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func (m *MTProto) Read(conn *net.TCPConn) (interface{}, error) {
	m.conn = conn
	return m.read()
}

func (m *MTProto) read() (interface{}, error) {
	var err error
	var n int
	var size int
	var data interface{}

	err = m.conn.SetReadDeadline(time.Now().Add(readDeadLine * time.Second))
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	b := make([]byte, 1)
	n, err = m.conn.Read(b)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	log.Println("here2")

	if b[0] == 0xef {
		log.Println(b)
		m.conn.Write([]byte("hi"))
		return nil, nil
	}

	if b[0] < 127 {
		size = int(b[0]) << 2
	} else {
		b := make([]byte, 3)
		n, err = m.conn.Read(b)
		if err != nil {
			return nil, err
		}
		size = (int(b[0]) | int(b[1])<<8 | int(b[2])<<16) << 2
	}
	log.Println("here3")

	log.Println(size)

	left := size
	buf := make([]byte, size)

	for left > 0 {
		n, err = m.conn.Read(buf[size-left:])
		if err != nil {
			return nil, err
		}
		left -= n
	}

	log.Println("here4")

	if size == 4 {
		return nil, fmt.Errorf("Server response error: %d", int32(binary.LittleEndian.Uint32(buf)))
	}

	dbuf := NewDecodeBuf(buf)

	authKeyHash := dbuf.Bytes(8)
	if binary.LittleEndian.Uint64(authKeyHash) == 0 {
		log.Println("here5 ")
		m.msgId = dbuf.Long()
		messageLen := dbuf.Int()
		if int(messageLen) != dbuf.size-20 {
			return nil, fmt.Errorf("Message len: %d (need %d)", messageLen, dbuf.size-20)
		}
		m.seqNo = 0

		data = dbuf.Object()
		if dbuf.err != nil {
			return nil, dbuf.err
		}

	} else {
		log.Println("here5A ")
		msgKey := dbuf.Bytes(16)
		encryptedData := dbuf.Bytes(dbuf.size - 24)
		aesKey, aesIV := generateAES(msgKey, m.authKey, true)
		x, err := doAES256IGEdecrypt(encryptedData, aesKey, aesIV)
		if err != nil {
			return nil, err
		}
		dbuf = NewDecodeBuf(x)
		_ = dbuf.Long() // salt
		_ = dbuf.Long() // session_id
		m.msgId = dbuf.Long()
		m.seqNo = dbuf.Int()
		messageLen := dbuf.Int()
		if int(messageLen) > dbuf.size-32 {
			return nil, fmt.Errorf("Message len: %d (need less than %d)", messageLen, dbuf.size-32)
		}
		if !bytes.Equal(sha1(dbuf.buf[0 : 32+messageLen])[4:20], msgKey) {
			return nil, errors.New("Wrong msg_key")
		}

		data = dbuf.Object()
		if dbuf.err != nil {
			return nil, dbuf.err
		}

	}
	log.Println(m.msgId)
	// mod := m.msgId & 3
	// if mod != 1 && mod != 3 {
	// 	return nil, fmt.Errorf("Wrong bits of message_id: %d", mod)
	// }

	return data, nil
}

func (m *MTProto) makeAuthKey() error {
	log.Println("makeAuthKey")
	var x []byte
	var err error
	var data interface{}

	// (send) req_pq
	nonceFirst := GenerateNonce(16)
	err = m.sendPacket(TL_req_pq{nonceFirst}, nil)
	if err != nil {
		return err
	}

	// (parse) resPQ
	data, err = m.read()
	if err != nil {
		return err
	}
	res, ok := data.(TL_resPQ)
	if !ok {
		return errors.New("Handshake: Need resPQ")
	}
	if !bytes.Equal(nonceFirst, res.Nonce) {
		return errors.New("Handshake: Wrong Nonce")
	}
	found := false
	for _, b := range res.Fingerprints {
		if uint64(b) == telegramPublicKey_FP {
			found = true
			break
		}
	}
	if !found {
		return errors.New("Handshake: No fingerprint")
	}

	// (encoding) p_q_inner_data
	p, q := splitPQ(res.Pq)
	nonceSecond := GenerateNonce(32)
	nonceServer := res.Server_nonce
	innerData1 := (TL_p_q_inner_data{res.Pq, p, q, nonceFirst, nonceServer, nonceSecond}).encode()

	x = make([]byte, 255)
	copy(x[0:], sha1(innerData1))
	copy(x[20:], innerData1)
	encryptedData1 := doRSAencrypt(x)
	// (send) req_DH_params
	err = m.sendPacket(TL_req_DH_params{nonceFirst, nonceServer, p, q, telegramPublicKey_FP, encryptedData1}, nil)
	if err != nil {
		return err
	}

	// (parse) server_DH_params_{ok, fail}
	data, err = m.read()
	if err != nil {
		return err
	}
	dh, ok := data.(TL_server_DH_params_ok)
	if !ok {
		return errors.New("Handshake: Need server_DH_params_ok")
	}
	if !bytes.Equal(nonceFirst, dh.Nonce) {
		return errors.New("Handshake: Wrong Nonce")
	}
	if !bytes.Equal(nonceServer, dh.Server_nonce) {
		return errors.New("Handshake: Wrong Server_nonce")
	}
	t1 := make([]byte, 48)
	copy(t1[0:], nonceSecond)
	copy(t1[32:], nonceServer)
	hash1 := sha1(t1)

	t2 := make([]byte, 48)
	copy(t2[0:], nonceServer)
	copy(t2[16:], nonceSecond)
	hash2 := sha1(t2)

	t3 := make([]byte, 64)
	copy(t3[0:], nonceSecond)
	copy(t3[32:], nonceSecond)
	hash3 := sha1(t3)

	tmpAESKey := make([]byte, 32)
	tmpAESIV := make([]byte, 32)

	copy(tmpAESKey[0:], hash1)
	copy(tmpAESKey[20:], hash2[0:12])

	copy(tmpAESIV[0:], hash2[12:20])
	copy(tmpAESIV[8:], hash3)
	copy(tmpAESIV[28:], nonceSecond[0:4])

	// (parse-thru) server_DH_inner_data
	decodedData, err := doAES256IGEdecrypt(dh.Encrypted_answer, tmpAESKey, tmpAESIV)
	if err != nil {
		return err
	}
	innerbuf := NewDecodeBuf(decodedData[20:])
	data = innerbuf.Object()
	if innerbuf.err != nil {
		return innerbuf.err
	}
	dhi, ok := data.(TL_server_DH_inner_data)
	if !ok {
		return errors.New("Handshake: Need server_DH_inner_data")
	}
	if !bytes.Equal(nonceFirst, dhi.Nonce) {
		return errors.New("Handshake: Wrong Nonce")
	}
	if !bytes.Equal(nonceServer, dhi.Server_nonce) {
		return errors.New("Handshake: Wrong Server_nonce")
	}

	_, g_b, g_ab := makeGAB(dhi.G, dhi.G_a, dhi.Dh_prime)
	m.authKey = g_ab.Bytes()
	if m.authKey[0] == 0 {
		m.authKey = m.authKey[1:]
	}
	m.authKeyHash = sha1(m.authKey)[12:20]
	t4 := make([]byte, 32+1+8)
	copy(t4[0:], nonceSecond)
	t4[32] = 1
	copy(t4[33:], sha1(m.authKey)[0:8])
	nonceHash1 := sha1(t4)[4:20]
	m.serverSalt = make([]byte, 8)
	copy(m.serverSalt, nonceSecond[:8])
	xor(m.serverSalt, nonceServer[:8])

	// (encoding) client_DH_inner_data
	innerData2 := (TL_client_DH_inner_data{nonceFirst, nonceServer, 0, g_b}).encode()
	x = make([]byte, 20+len(innerData2)+(16-((20+len(innerData2))%16))&15)
	copy(x[0:], sha1(innerData2))
	copy(x[20:], innerData2)
	encryptedData2, err := doAES256IGEencrypt(x, tmpAESKey, tmpAESIV)

	// (send) set_client_DH_params
	err = m.sendPacket(TL_set_client_DH_params{nonceFirst, nonceServer, encryptedData2}, nil)
	if err != nil {
		return err
	}

	// (parse) dh_gen_{ok, Retry, fail}
	data, err = m.read()
	if err != nil {
		return err
	}
	dhg, ok := data.(TL_dh_gen_ok)
	if !ok {
		return errors.New("Handshake: Need dh_gen_ok")
	}
	if !bytes.Equal(nonceFirst, dhg.Nonce) {
		return errors.New("Handshake: Wrong Nonce")
	}
	if !bytes.Equal(nonceServer, dhg.Server_nonce) {
		return errors.New("Handshake: Wrong Server_nonce")
	}
	if !bytes.Equal(nonceHash1, dhg.New_nonce_hash1) {
		return errors.New("Handshake: Wrong New_nonce_hash1")
	}

	// (all ok)
	err = m.saveData()
	if err != nil {
		return err
	}

	return nil
}

func ReadData(conn net.Conn, cd *CacheData) (interface{}, error) {
	var err error
	var n int
	var size int
	// var msgId int64
	var data interface{}

	err = conn.SetReadDeadline(time.Now().Add(readDeadLine * time.Second))
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	b := make([]byte, 1)
	n, err = conn.Read(b)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	if b[0] == 0xef {
		log.Println(b)
		return nil, nil
	}

	if b[0] < 127 {
		size = int(b[0]) << 2
	} else {
		b := make([]byte, 3)
		n, err = conn.Read(b)
		if err != nil {
			return nil, err
		}
		size = (int(b[0]) | int(b[1])<<8 | int(b[2])<<16) << 2
	}

	left := size
	buf := make([]byte, size)

	for left > 0 {
		n, err = conn.Read(buf[size-left:])
		if err != nil {
			return nil, err
		}
		left -= n
	}

	if size == 4 {
		return nil, fmt.Errorf("Server response error: %d", int32(binary.LittleEndian.Uint32(buf)))
	}

	dbuf := NewDecodeBuf(buf)
	// spew.Dump(buf)

	authKeyHash := dbuf.Bytes(8)
	if binary.LittleEndian.Uint64(authKeyHash) == 0 {
		cd.MsgID = dbuf.Long()
		messageLen := dbuf.Int()
		if int(messageLen) != dbuf.size-20 {
			return nil, fmt.Errorf("Message len: %d (need %d)", messageLen, dbuf.size-20)
		}
		cd.SeqNo = 0

		data = dbuf.Object()
		if dbuf.err != nil {
			return nil, dbuf.err
		}

	} else {
		msgKey := dbuf.Bytes(16)
		encryptedData := dbuf.Bytes(dbuf.size - 24)
		aesKey, aesIV := generateAES(msgKey, cd.AuthKey, false)

		x, err := doAES256IGEdecrypt(encryptedData, aesKey, aesIV)
		if err != nil {
			return nil, err
		}
		dbuf = NewDecodeBuf(x)

		_ = dbuf.Long()            // salt
		cd.SessionID = dbuf.Long() // session_id
		cd.MsgID = dbuf.Long()
		log.Println("msgid:", cd.MsgID)
		cd.SeqNo = dbuf.Int()
		log.Println("seq no:", cd.SeqNo)
		messageLen := dbuf.Int()
		log.Println("msg len", messageLen)
		if int(messageLen) > dbuf.size-32 {
			return nil, fmt.Errorf("Message len: %d (need less than %d)", messageLen, dbuf.size-32)
		}
		if !bytes.Equal(sha1(dbuf.buf[0 : 32+messageLen])[4:20], msgKey) {
			return nil, errors.New("Wrong msg_key")
		}

		data = dbuf.Object()
		if dbuf.err != nil {
			return nil, dbuf.err
		}

	}
	log.Println(cd.MsgID)
	// mod := m.msgId & 3
	// if mod != 1 && mod != 3 {
	// 	return nil, fmt.Errorf("Wrong bits of message_id: %d", mod)
	// }

	return data, nil
}

type HttpData struct {
	Data      interface{}
	CacheData CacheData
}

func ReadHttpData(rData []byte, sessions map[string]*CacheData) (*HttpData, error) {
	var size int
	hData := new(HttpData)

	if rData[0] == 0xef {
		return nil, nil
	}

	if rData[0] < 127 {
		size = int(rData[0]) << 2
	} else {
		b := rData[1:4]
		size = (int(b[0]) | int(b[1])<<8 | int(b[2])<<16) << 2
	}

	buf := rData

	if size == 4 {
		return nil, fmt.Errorf("Server response error: %d", int32(binary.LittleEndian.Uint32(buf)))
	}

	dbuf := NewDecodeBuf(buf)
	// spew.Dump(buf)

	authKeyHash := dbuf.Bytes(8)
	if binary.LittleEndian.Uint64(authKeyHash) == 0 {

		_ = dbuf.Long()
		messageLen := dbuf.Int()
		if int(messageLen) != dbuf.size-20 {
			return nil, fmt.Errorf("Message len: %d (need %d)", messageLen, dbuf.size-20)
		}
		_ = 0

		hData.Data = dbuf.Object()
		if dbuf.err != nil {
			return nil, dbuf.err
		}

	} else {
		authKeySha1 := fmt.Sprintf("%x", sha1(authKeyHash))

		rCon := getRedisClient()
		b, _ := rCon.Get("authkey:" + authKeySha1).Bytes()
		var cd CacheData
		json.Unmarshal(b, &cd)
		// hData.CacheData = cd

		msgKey := dbuf.Bytes(16)
		encryptedData := dbuf.Bytes(dbuf.size - 24)
		authKeyByte := cd.AuthKey
		aesKey, aesIV := generateAES(msgKey, authKeyByte, false)

		x, err := doAES256IGEdecrypt(encryptedData, aesKey, aesIV)
		if err != nil {
			return nil, err
		}
		dbuf = NewDecodeBuf(x)

		_ = dbuf.Long()            // salt
		cd.SessionID = dbuf.Long() // session_id
		cd.MsgID = dbuf.Long()
		// log.Println("msgid:", cd.MsgID)
		_ = dbuf.Int()
		// log.Println("seq no:", cd.SeqNo)
		messageLen := dbuf.Int()
		// log.Println("msg len", messageLen)
		if int(messageLen) > dbuf.size-32 {
			return nil, fmt.Errorf("Message len: %d (need less than %d)", messageLen, dbuf.size-32)
		}
		if !bytes.Equal(sha1(dbuf.buf[0 : 32+messageLen])[4:20], msgKey) {
			return nil, errors.New("Wrong msg_key")
		}

		hData.CacheData = cd

		hData.Data = dbuf.Object()
		if dbuf.err != nil {
			return nil, dbuf.err
		}

	}

	return hData, nil
}

var redisClient *redis.Client

func getRedisClient() *redis.Client {
	if redisClient != nil {
		return redisClient
	}
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return redisClient
}

func EncodeTL(msg TL) []byte {
	obj := msg.encode()
	return obj
}

func EncodeInterface(msg interface{}) []byte {
	msgTl := msg.(TL)
	obj := msgTl.encode()
	return obj
}

func MakePacket(msg TL) ([]byte, error) {
	// log.Println("make packet ", msg)
	obj := msg.encode()
	x := NewEncodeBuf(256)

	// padding for tcpsize
	x.Int(0)

	x.Long(0)
	msgID := GenerateMessageId()
	log.Println("message id:", msgID)
	x.Long(msgID)
	x.Int(int32(len(obj)))
	x.Bytes(obj)

	// minus padding
	size := len(x.buf)/4 - 1

	log.Println("size is:", size, len(x.buf))

	if size < 127 {
		x.buf[3] = byte(size)
		x.buf = x.buf[3:]
	} else {
		binary.LittleEndian.PutUint32(x.buf, uint32(size<<8|127))
	}

	// log.Println("b,", x.buf[0])

	return x.buf, nil
}

func MakePacketHttp(msg TL, cd *CacheData) ([]byte, error) {
	// log.Println("make packet ", msg)
	obj := msg.encode()
	x := NewEncodeBuf(256)

	log.Println("make http packet")
	spew.Dump(msg)

	// padding for tcpsize
	// x.Int(0)

	if cd != nil {
		if cd.Encrypted {
			needAck := true
			switch msg.(type) {
			case TL_ping, TL_msgs_ack:
				needAck = false
			}
			z := NewEncodeBuf(256)
			newMsgId := GenerateMessageId()

			z.Bytes(cd.ServerSalt)
			z.Long(cd.SessionID)
			z.Long(newMsgId)
			if needAck {
				z.Int(cd.SeqNo | 1)
			} else {
				z.Int(cd.SeqNo)
			}
			z.Int(int32(len(obj)))
			z.Bytes(obj)

			msgKey := sha1(z.buf)[4:20]
			aesKey, aesIV := generateAES(msgKey, cd.AuthKey, true)

			y := make([]byte, len(z.buf)+((16-(len(obj)%16))&15))
			copy(y, z.buf)

			encryptedData, err := doAES256IGEencrypt(y, aesKey, aesIV)
			if err != nil {
				return nil, err
			}

			cd.SeqNo += 2

			x.Bytes(cd.AuthKeyHash)
			x.Bytes(msgKey)
			x.Bytes(encryptedData)

		}
	} else {

		x.Long(0)
		msgID := GenerateMessageId()
		log.Println("message id:", msgID)
		x.Long(msgID)
		x.Int(int32(len(obj)))
		x.Bytes(obj)
	}

	return x.buf, nil
}

func MakingPacket(msg TL, cd *CacheData) ([]byte, error) {
	log.Println("making packet ", msg, reflect.TypeOf(msg))
	obj := msg.encode()
	log.Println("data of packet")
	// spew.Dump(obj)
	x := NewEncodeBuf(256)

	// padding for tcpsize
	x.Int(0)

	if cd.Encrypted {
		needAck := true
		switch msg.(type) {
		case TL_ping, TL_msgs_ack:
			needAck = false
		}
		z := NewEncodeBuf(256)
		newMsgId := GenerateMessageId()

		z.Bytes(cd.ServerSalt)
		z.Long(cd.SessionID)
		z.Long(newMsgId)
		if needAck {
			z.Int(cd.SeqNo | 1)
		} else {
			z.Int(cd.SeqNo)
		}
		z.Int(int32(len(obj)))
		z.Bytes(obj)

		msgKey := sha1(z.buf)[4:20]
		aesKey, aesIV := generateAES(msgKey, cd.AuthKey, true)

		y := make([]byte, len(z.buf)+((16-(len(obj)%16))&15))
		copy(y, z.buf)

		encryptedData, err := doAES256IGEencrypt(y, aesKey, aesIV)
		if err != nil {
			return nil, err
		}

		cd.SeqNo += 2

		x.Bytes(cd.AuthKeyHash)
		x.Bytes(msgKey)
		x.Bytes(encryptedData)

	} else {
		x.Long(0)
		x.Long(GenerateMessageId())
		x.Int(int32(len(obj)))
		x.Bytes(obj)
	}

	// minus padding
	size := len(x.buf)/4 - 1

	if size < 127 {
		x.buf[3] = byte(size)
		x.buf = x.buf[3:]
	} else {
		binary.LittleEndian.PutUint32(x.buf, uint32(size<<8|127))
	}

	return x.buf, nil

}
