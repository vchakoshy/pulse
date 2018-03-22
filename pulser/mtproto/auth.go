package mtproto

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func (m *MTProto) AuthSendCode(phonenumber string) (*TL_auth_sentCode, error) {
	var authSentCode TL_auth_sentCode
	flag := true
	for flag {
		resp := make(chan response, 1)
		m.queueSend <- packetToSend{
			msg: TL_auth_sendCode{
				Allow_flashcall: false,
				Phone_number:    phonenumber,
				Current_number:  TL_boolTrue{},
				Api_id:          m.appConfig.Id,
				Api_hash:        m.appConfig.Hash,
			},
			resp: resp,
		}
		x := <-resp

		if x.err != nil {
			// TODO: Maybe there are different ways to do it
			// MTProto connected to new DC(see handleRPCError), trying to get data again
			if strings.Contains(x.err.Error(), strconv.Itoa(errorSeeOther)) {
				continue
			}
			return nil, x.err
		}

		switch x.data.(type) {
		case TL_auth_sentCode:
			authSentCode = x.data.(TL_auth_sentCode)
			flag = false
		default:
			return nil, fmt.Errorf("Got: %T", x)
		}
	}

	return &authSentCode, nil
}

func (m *MTProto) Auth(phonenumber string) error {
	var code string

	/* Authenticate */
	authSentCode, err := m.AuthSendCode(phonenumber)
	if err != nil {
		return err
	}

	if !authSentCode.Phone_registered {
		err := errors.New("Cannot sign in: Phone isn't registered")
		return err
	}

	fmt.Printf("Enter code: ")
	fmt.Scanf("%s", &code)

	auth, err := m.AuthSignIn(phonenumber, code, authSentCode.Phone_code_hash)
	if err != nil {
		return err
	}
	userSelf := auth.User.(TL_user)
	fmt.Printf("Signed in: Id %d name <%s %s>\n", userSelf.Id, userSelf.First_name, userSelf.Last_name)
	return nil
}

func (m *MTProto) AuthSignIn(phoneNumber, phoneCode, phoneCodeHash string) (*TL_auth_authorization, error) {
	if phoneNumber == "" || phoneCode == "" || phoneCodeHash == "" {
		return nil, errors.New("MRProto::AuthSignIn one of function parameters is empty")
	}

	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_auth_signIn{
			Phone_number:    phoneNumber,
			Phone_code_hash: phoneCodeHash,
			Phone_code:      phoneCode,
		},
		resp: resp,
	}
	x := <-resp
	if x.err != nil {
		return nil, x.err
	}

	auth, ok := x.data.(TL_auth_authorization)

	if !ok {
		return nil, fmt.Errorf("RPC: %#v", x)
	}

	return &auth, nil
}

func (m *MTProto) AuthLogOut() (bool, error) {
	var result bool
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg:  TL_auth_logOut{},
		resp: resp,
	}
	x := <-resp
	if x.err != nil {
		return result, x.err
	}

	result, err := ToBool(x.data)
	if err != nil {
		return result, err
	}

	return result, err
}

func (m *MTProto) GetConfig() error {
	// path := "./dc_list.json"
	// if m.started {
	// 	err := m.GetDC(path)
	// 	if err == nil {
	// 		return nil
	// 	}
	// }
	resp := make(chan response, 1)

	m.queueSend <- packetToSend{
		msg: TL_invokeWithLayer{
			Layer: 65,
			Query: TL_initConnection{
				Api_id:         m.appConfig.Id,
				Device_model:   m.appConfig.DeviceModel,
				System_version: m.appConfig.SystemVersion,
				App_version:    m.appConfig.Version,
				Lang_code:      m.appConfig.Language,
				Query:          TL_help_getConfig{},
			},
		},
		resp: resp,
	}
	x := <-resp
	if x.err != nil {
		return x.err
	}

	switch x.data.(type) {
	case TL_config:
		data := x.data.(TL_config)

		log.Println("dc_id: ", data.This_dc)
		m.CurrentDC = data.This_dc
		m.dclist = make(map[int32]string, 5)
		for _, v := range data.Dc_options {
			v := v.(TL_dcOption)
			if v.Ipv6 {
				continue
			}
			if v.Media_only {
				continue
			}
			m.dclist[v.Id] = fmt.Sprintf("%s:%d", v.Ip_address, v.Port)
		}
		log.Println(m.dclist)
	default:
		return fmt.Errorf("Connection error: got: %T", x)
	}
	log.Println("getConfig")
	return nil
}

func (m *MTProto) GetDC(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	err = json.Unmarshal(b, &m.dclist)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	if len(m.dclist) == 0 {
		return fmt.Errorf("DCs list is empty")
	}
	log.Println("Get DCs list")
	return nil
}

func (m *MTProto) SaveDC(path string, DCList map[int32]string) error {
	jsonFile, err := os.Create(path)
	defer jsonFile.Close()
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	jsonData, err := json.Marshal(DCList)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	jsonFile.Write(jsonData)
	log.Println("Save DCs list")
	return nil
}

func (m *MTProto) AuthImportAuthorization(id int32, bytes []byte) (*TL_auth_authorization, error) {
	log.Println("AuthImportAuthorization")
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_auth_importAuthorization{
			Id:    id,
			Bytes: bytes,
		},
		resp: resp,
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(10 * time.Second)
		timeout <- true
	}()
	select {
	case x := <-resp:
		if x.err != nil {
			log.Println(x.err.Error())
			return nil, x.err
		}
		auth := x.data.(TL_auth_authorization)
		return &auth, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}
}

func (m *MTProto) AuthExportAuthorization(dc_id int32) (*TL_auth_exportedAuthorization, error) {
	log.Println("AuthExportAuthorization")
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_auth_exportAuthorization{
			Dc_id: dc_id,
		},
		resp: resp,
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(10 * time.Second)
		timeout <- true
	}()
	select {
	case x := <-resp:
		if x.err != nil {
			log.Println(x.err.Error())
			return nil, x.err
		}
		auth := x.data.(TL_auth_exportedAuthorization)
		return &auth, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}
}
