package mtproto

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
)

type MTProto struct {
	Addr         string
	conn         *net.TCPConn
	f            *os.File
	queueSend    chan packetToSend
	StopRoutines chan struct{}
	allDone      sync.WaitGroup

	authKey     []byte
	authKeyHash []byte
	serverSalt  []byte
	encrypted   bool
	sessionId   int64

	mutex        *sync.Mutex
	lastSeqNo    int32
	msgsIdToAck  map[int64]packetToSend
	msgsIdToResp map[int64]chan response
	seqNo        int32
	msgId        int64
	IsMigrate    bool
	IsReconnect  bool
	IsConnected  bool
	HasErr       bool
	CurrentDC    int32

	appConfig Configuration

	dclist       map[int32]string
	CurUserPhone string
}

type packetToSend struct {
	msg  TL
	resp chan response
}

type response struct {
	data TL
	err  error
}

type Configuration struct {
	Id            int32
	Hash          string
	Version       string
	DeviceModel   string
	SystemVersion string
	Language      string
}

// API Errors
const (
	errorSeeOther     = 303
	errorBadRequest   = 400
	errorUnauthorized = 401
	errorForbidden    = 403
	errorNotFound     = 404
	errorFlood        = 420
	errorInternal     = 500
)

const appConfigError = "App configuration error: %s"

// Current API Layer Version
const layer = 65

func NewConfiguration(id int32, hash, version, deviceModel, systemVersion, language string) (*Configuration, error) {
	appConfig := new(Configuration)

	if id == 0 || hash == "" || version == "" {
		return nil, fmt.Errorf(appConfigError, "Fields Id, Hash or Version are empty")
	}
	appConfig.Id = id
	appConfig.Hash = hash
	appConfig.Version = version

	appConfig.DeviceModel = deviceModel
	if deviceModel == "" {
		appConfig.DeviceModel = "Unknown"
	}

	appConfig.SystemVersion = systemVersion
	if systemVersion == "" {
		appConfig.SystemVersion = runtime.GOOS + "/" + runtime.GOARCH
	}

	appConfig.Language = language
	if language == "" {
		appConfig.Language = "en"
	}

	return appConfig, nil
}

func (appConfig Configuration) Check() error {
	if appConfig.Id == 0 || appConfig.Hash == "" || appConfig.Version == "" {
		return fmt.Errorf(appConfigError, "Configuration.Id, Configuration.Hash or Configuration.Version are empty")
	}

	if appConfig.DeviceModel == "" {
		return fmt.Errorf(appConfigError, "Configuration.DeviceModel is empty")
	}

	if appConfig.SystemVersion == "" {
		return fmt.Errorf(appConfigError, "Configuration.SystemVersion is empty")
	}

	if appConfig.Language == "" {
		return fmt.Errorf(appConfigError, "Configuration.Language is empty")
	}

	return nil
}

func NewMTProto(newSession bool, phoneNumber, authKeyPath string) (*MTProto, error) {
	var err error

	appConfig, err := NewConfiguration(appId, appHash, "0.0.4", "PC,Linux", "0.0.2", "en")
	if err != nil {
		return nil, err
	}

	err = appConfig.Check()
	if err != nil {
		return nil, err
	}

	m := new(MTProto)
	m.appConfig = *appConfig
	m.CurUserPhone = phoneNumber

	// filename := fmt.Sprintf("%s/%s_%s", authKeyPath, "authkey", phoneNumber)
	// m.f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	// if err != nil {
	// 	return nil, err
	// }

	rand.Seed(time.Now().UnixNano())
	m.sessionId = rand.Int63()

	if newSession {
		m.Addr = serverAddr
		m.encrypted = false
		return m, nil
	}

	err = m.readData()
	if err == nil {
		m.encrypted = true
	} else {
		m.Addr = serverAddr
		m.encrypted = false
	}
	m.IsConnected = false
	m.IsReconnect = false
	m.HasErr = false

	return m, nil
}

func (m *MTProto) Connect() error {
	log.Println("Connect")
	var err error
	var tcpAddr *net.TCPAddr

	// connect
	tcpAddr, err = net.ResolveTCPAddr("tcp", m.Addr)
	if err != nil {
		return err
	}

	m.conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	}

	// Packet Length is encoded by a single byte (see: https://core.telegram.org/mtproto)
	_, err = m.conn.Write([]byte{0xef})
	if err != nil {
		return err
	}

	// get new authKey if need
	if !m.encrypted {
		err = m.makeAuthKey()
		if err != nil {
			return err
		}
	}
	// start goroutines
	m.queueSend = make(chan packetToSend, 64)
	m.StopRoutines = make(chan struct{})
	m.allDone = sync.WaitGroup{}
	m.msgsIdToAck = make(map[int64]packetToSend)
	m.msgsIdToResp = make(map[int64]chan response)
	m.mutex = &sync.Mutex{}
	go m.sendRoutine()
	go m.readRoutine()

	err = m.GetConfig()
	if err != nil {
		return err
	}

	// start keep alive ping
	go m.pingRoutine()
	m.IsConnected = true
	return nil
}

func (m *MTProto) Disconnect() error {
	// stop ping, send and read routine by closing channel StopRoutines
	close(m.StopRoutines)

	// Wait until all goroutines stopped
	m.allDone.Wait()

	// close send queue
	close(m.queueSend)

	m.IsConnected = false
	// close connection
	err := m.conn.Close()
	if err != nil {
		return err
	}

	return nil
}

func (m *MTProto) Reconnect(newaddr string) error {
	log.Println("Reconnect")
	m.IsReconnect = true
	if !m.IsMigrate {
		err := m.Disconnect()
		if err != nil {
			return err
		}
	} else {
		m.IsMigrate = false
	}
	// renew connection
	m.encrypted = true
	if m.Addr != newaddr {
		m.encrypted = false
	}
	m.Addr = newaddr
	err := m.Connect()
	return err
}

func (m *MTProto) pingRoutine() {
	m.allDone.Add(1)
	defer func() { m.allDone.Done() }()
	for {
		select {
		case <-m.StopRoutines:
			log.Println("get stop routine")
			return
		case <-time.After(60 * time.Second):
			m.queueSend <- packetToSend{TL_ping{0xCADACADA}, nil}
		}
	}
}

func (m *MTProto) sendRoutine() {
	m.allDone.Add(1)
	defer func() { m.allDone.Done() }()
	for {
		select {
		case <-m.StopRoutines:
			log.Println("get stop routine")
			return
		case x := <-m.queueSend:
			err := m.sendPacket(x.msg, x.resp)
			if err != nil {
				log.Println("SendRoutine:", err.Error())
				// m.Close()
				return
			}
		}
	}
}

// func (m *MTProto) Close() {

// 	// close(m.queueSend)
// 	m.StopRoutines <- struct{}{}
// 	m.StopRoutines <- struct{}{}
// 	m.StopRoutines <- struct{}{}
// }

func (m *MTProto) readRoutine() {
	m.allDone.Add(1)
	defer func() { m.allDone.Done() }()
	for {
		// Run async wait for data from server
		ch := make(chan interface{}, 1)
		go func(ch chan<- interface{}) {
			data, err := m.read()
			if err != nil {
				log.Println("ReadRoutine: ", err.Error())
				// m.Close()
				m.HasErr = true
				return
			}
			ch <- data
		}(ch)

		select {
		case <-m.StopRoutines:
			log.Println("get stop routine")
			return
		case data := <-ch:
			m.process(m.msgId, m.seqNo, data)
		}
	}
}

func (m *MTProto) process(msgId int64, seqNo int32, data interface{}) interface{} {
	switch data.(type) {
	case TL_msg_container:
		data := data.(TL_msg_container).Items
		for _, v := range data {
			m.process(v.Msg_id, v.Seq_no, v.Data)
		}

	case TL_bad_server_salt:
		data := data.(TL_bad_server_salt)
		m.serverSalt = data.New_server_salt
		_ = m.saveData()
		m.mutex.Lock()
		defer m.mutex.Unlock()
		for k, v := range m.msgsIdToAck {
			delete(m.msgsIdToAck, k)
			m.queueSend <- v
		}

	case TL_new_session_created:
		data := data.(TL_new_session_created)
		m.serverSalt = data.Server_salt
		_ = m.saveData()

	case TL_ping:
		data := data.(TL_ping)
		m.queueSend <- packetToSend{TL_pong{msgId, data.Ping_id}, nil}

	case TL_pong:
		// ignore

	case TL_msgs_ack:
		data := data.(TL_msgs_ack)
		m.mutex.Lock()
		defer m.mutex.Unlock()
		for _, v := range data.MsgIds {
			delete(m.msgsIdToAck, v)
		}

	case TL_rpc_result:
		data := data.(TL_rpc_result)
		x := m.process(msgId, seqNo, data.Obj)
		m.mutex.Lock()
		defer m.mutex.Unlock()
		v, ok := m.msgsIdToResp[data.Req_msg_id]
		if ok {
			var resp response
			rpcError, ok := x.(TL_rpc_error)
			if ok {
				resp.err = m.handleRPCError(rpcError)
			}
			resp.data = x.(TL)
			v <- resp
			close(v)
			delete(m.msgsIdToResp, data.Req_msg_id)
		}
		delete(m.msgsIdToAck, data.Req_msg_id)

	default:
		return data
	}

	// TODO: Check why I should do this
	if (seqNo & 1) == 1 {
		m.queueSend <- packetToSend{TL_msgs_ack{[]int64{msgId}}, nil}
	}

	return nil
}

func (m *MTProto) handleRPCError(rpcError TL_rpc_error) error {

	switch rpcError.Error_code {
	case errorSeeOther:
		var newDc int32
		n, _ := fmt.Sscanf(rpcError.Error_message, "PHONE_MIGRATE_%d", &newDc)
		if n != 1 {
			n, _ := fmt.Sscanf(rpcError.Error_message, "NETWORK_MIGRATE_%d", &newDc)
			if n != 1 {
				return fmt.Errorf("RPC error_string: %s", rpcError.Error_message)
			}
		}
		newDcAddr, ok := m.dclist[newDc]
		if !ok {
			return fmt.Errorf("Wrong DC index: %d", newDc)
		}
		m.IsMigrate = true
		log.Println("migrate server")
		err := m.Reconnect(newDcAddr)
		if err != nil {
			return err
		}
		return fmt.Errorf(rpcError.Error_message)

	case errorFlood:
		var wait int32
		fmt.Sscanf(rpcError.Error_message, "FLOOD_WAIT_%d", &wait)
		log.Println("Sleep for ", wait, " second ", m.CurUserPhone, rpcError.Error_message)
		// time.Sleep(time.Duration(wait) * time.Second)
		return fmt.Errorf(rpcError.Error_message)

	case errorBadRequest, errorInternal, errorUnauthorized:
		return fmt.Errorf(rpcError.Error_message)

	default:
		return fmt.Errorf(rpcError.Error_message)
	}
}

// Save session
func (m *MTProto) saveData() (err error) {
	m.encrypted = true

	b := NewEncodeBuf(1024)
	b.StringBytes(m.authKey)
	b.StringBytes(m.authKeyHash)
	b.StringBytes(m.serverSalt)
	b.String(m.Addr)

	err = m.f.Truncate(0)
	if err != nil {
		return err
	}

	_, err = m.f.WriteAt(b.buf, 0)
	if err != nil {
		return err
	}

	return nil
}

// Load session
func (m *MTProto) readData() (err error) {
	b := make([]byte, 1024*4)
	n, err := m.f.ReadAt(b, 0)
	if n <= 0 {
		return errors.New("New session")
	}

	d := NewDecodeBuf(b)
	m.authKey = d.StringBytes()
	m.authKeyHash = d.StringBytes()
	m.serverSalt = d.StringBytes()
	m.Addr = d.String()

	if d.err != nil {
		return d.err
	}

	return nil
}
