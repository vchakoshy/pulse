package mtproto

import (
	"log"
	"time"
)

func (m *MTProto) UsersGetFullUsers(id TL) (*TL_userFull, error) {
	var user TL_userFull
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_users_getFullUser{
			Id: id,
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
			return nil, x.err
		}
		user = x.data.(TL_userFull)
		m.CurUserPhone = user.User.(TL_user).Phone
		return &user, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}
}

func (m *MTProto) AccountChangeStatus(offline bool) (TL, error) {

	var x response

	resp := make(chan response, 1)
	if offline {
		m.queueSend <- packetToSend{
			msg: TL_account_updateStatus{
				Offline: TL_boolTrue{},
			},
			resp: resp,
		}
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(10 * time.Second)
			timeout <- true
		}()
		select {
		case x = <-resp:
			if x.err != nil {
				log.Println(x.err.Error())
				return nil, x.err
			}

			return x.data, nil
		case <-timeout:
			log.Println("time out on response")
			return nil, ErrTelegramTimeOut
		}

	} else {
		m.queueSend <- packetToSend{
			msg: TL_account_updateStatus{
				Offline: TL_boolFalse{},
			},
			resp: resp,
		}
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(10 * time.Second)
			timeout <- true
		}()

		select {
		case x = <-resp:
			if x.err != nil {
				log.Println(x.err.Error())
				return nil, x.err
			}
			return x.data, nil
		case <-timeout:
			log.Println("time out on response")
			return nil, ErrTelegramTimeOut
		}
	}
}

func (m *MTProto) SendProfilePhoto(photoId int64, hash int64, objectId string, randomId, BotAccessHash int64, BotId int32) error {

	inputPhoto := TL_inputPhoto{Id: photoId, Access_hash: hash}
	media := TL_inputMediaPhoto{Id: inputPhoto, Caption: objectId}

	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_messages_sendMedia{
			Peer:         TL_inputPeerUser{User_id: BotId, Access_hash: BotAccessHash},
			Media:        media,
			Random_id:    randomId,
			Reply_markup: TL_inputBotInlineMessageID{}},
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
			return x.err
		}
		return nil
	case <-timeout:
		log.Println("time out on response")
		return ErrTelegramTimeOut
	}
}
