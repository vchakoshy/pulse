package mtproto

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"
)

func (m *MTProto) MessagesGetHistory(peer TL, offsetId, offsetDate, addOffset, limit, maxID, minId int32) (TL, error) {
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_messages_getHistory{
			Peer:        peer,
			Offset_id:   offsetId,
			Offset_date: offsetDate,
			Add_offset:  addOffset,
			Limit:       limit,
			Max_id:      maxID,
			Min_id:      minId,
		},
		resp: resp,
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(50 * time.Second)
		timeout <- true
	}()

	select {
	case x := <-resp:
		if x.err != nil {
			return nil, x.err
		}
		return x.data, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}
}

func (m *MTProto) GetDialogs(excludePinned bool, offsetDate, offsetId int32, offsetPeer TL, limit int32) (TL, error) {
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_messages_getDialogs{
			Exclude_pinned: excludePinned,
			Offset_date:    offsetDate,
			Offset_id:      offsetId,
			Offset_peer:    offsetPeer,
			Limit:          limit,
		},
		resp: resp,
	}
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(50 * time.Second)
		timeout <- true
	}()
	select {
	case x := <-resp:
		if x.err != nil {
			return nil, x.err
		}
		return x.data, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}
}

func (m *MTProto) MessagesSendMessage(no_webpage, silent, background, clear_draft bool, peer TL,
	reply_to_msg_id int32, message string, random_id int64, reply_markup TL, entities []TL) (*TL, error) {
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_messages_sendMessage{
			No_webpage:      no_webpage,
			Silent:          silent,
			Background:      background,
			Clear_draft:     clear_draft,
			Peer:            peer,
			Reply_to_msg_id: reply_to_msg_id,
			Message:         message,
			Random_id:       random_id,
			Reply_markup:    reply_markup,
			Entities:        entities,
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
		return &x.data, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}
}

func (m *MTProto) MessagesForwardMessage(Silent, Background, With_my_score bool, From_peer TL, id []int32, Random_id []int64, to_peer TL) (TL, error) {
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_messages_forwardMessages{
			0,
			Silent,
			Background,
			With_my_score,
			From_peer,
			id,
			Random_id,
			to_peer,
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
		return x.data, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}
}

func (m *MTProto) GetMessage(msgId, channelId, hash string) (Message, error) {
	resp := make(chan response, 1)
	list := make([]int32, 0)
	msg := Message{}
	var res response

	msgID, _ := strconv.Atoi(msgId)
	list = append(list, int32(msgID))

	channelID, _ := strconv.Atoi(channelId)
	accessHash, _ := strconv.ParseInt(hash, 10, 64)
	channel := TL_inputChannel{int32(channelID), accessHash}

	m.queueSend <- packetToSend{
		msg:  TL_channels_getMessages{Channel: channel, Id: list},
		resp: resp,
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()
	select {
	case x := <-resp:
		if x.err != nil {
			return msg, x.err
		}
		res = x
	case <-timeout:
		log.Println("time out on response")
		return msg, ErrTelegramTimeOut
	}

	channelMessage, ok := res.data.(TL_messages_channelMessages)
	if !ok {
		err := errors.New("error")
		return msg, err
	}

	switch channelMessage.Messages[0].(type) {
	case TL_message:
		chMsg := channelMessage.Messages[0].(TL_message)
		uniqeID := fmt.Sprintf("telegram.com/%s/%s", channelId, msgId)
		data := []byte(uniqeID)

		msg.ID = fmt.Sprintf("%x", md5.Sum(data))
		msg.Date = chMsg.Date
		msg.Text = chMsg.Message
		msg.ViewCount = chMsg.Views
		msg.Media = chMsg.Media
	default:
		err := errors.New("not found")
		return msg, err
	}
	return msg, nil
}

func (m *MTProto) GetPosts(msgsId []int32, channelId int32, hash int64) []TL_message {
	resp := make(chan response, 1)
	posts := make([]TL_message, 0)
	channel := TL_inputChannel{channelId, hash}

	m.queueSend <- packetToSend{
		msg:  TL_channels_getMessages{Channel: channel, Id: msgsId},
		resp: resp,
	}
	x := <-resp
	if x.err != nil {
		log.Println(x.err.Error())
		return posts
	}

	channelMessage, ok := x.data.(TL_messages_channelMessages)
	if !ok {
		return posts
	}

	messages := channelMessage.Messages
	for _, v := range messages {

		switch v.(type) {
		case TL_message:
			// message := m.GetMessageObject(v.(TL_message), channelId)
			posts = append(posts, v.(TL_message))
		default:
			// log.Println(v, "id")
			continue
		}

	}

	return posts
}

func (m *MTProto) SendMedia(media TL, randomId, botAccessHash int64, BotId int32) error {
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_messages_sendMedia{
			Peer:         TL_inputPeerUser{User_id: BotId, Access_hash: botAccessHash},
			Media:        media,
			Random_id:    randomId,
			Reply_markup: TL_replyKeyboardHide{}},
		resp: resp,
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(30 * time.Second)
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
func (m *MTProto) MessageSendMedia(msg Message, objectId string, randomId, botAccessHash int64, BotId int32) (err error) {
	var (
		media TL
	)

	switch msg.Media.(type) {
	case TL_messageMediaPhoto:
		msgPhoto := msg.Media.(TL_messageMediaPhoto).Photo
		switch msgPhoto.(type) {
		case TL_photo:
			photo := msgPhoto.(TL_photo)
			fileId := photo.Id
			hash := photo.Access_hash
			inputPhoto := TL_inputPhoto{Id: fileId, Access_hash: hash}
			media = TL_inputMediaPhoto{Id: inputPhoto, Caption: objectId}
		}

	case TL_messageMediaDocument:
		msgMedia := msg.Media.(TL_messageMediaDocument).Document
		switch msgMedia.(type) {
		case TL_document:
			doc := msgMedia.(TL_document)
			if doc.Mime_type != "video/mp4" {
				break
			}
			fileId := doc.Id
			hash := doc.Access_hash
			inputDoc := TL_inputDocument{Id: fileId, Access_hash: hash}
			media = TL_inputMediaDocument{Id: inputDoc, Caption: objectId}
		}

	}

	if media == nil {
		return nil
	}

	if botAccessHash == 0 {
		log.Println("bot access hash nil")
		return nil
	}
	err = m.SendMedia(media, randomId, botAccessHash, BotId)
	if err != nil {
		return err
	}
	return nil
}

func (m *MTProto) UpdateVeiwsCount(ids []string, channelID, accessHash string) (TL, error) {
	resp := make(chan response, 1)

	// ch, _ := m.GetChannelFromUsername("basket65")
	cid, _ := strconv.Atoi(channelID)
	ach, _ := strconv.Atoi(accessHash)
	peer := TL_inputPeerChannel{Channel_id: int32(cid), Access_hash: int64(ach)}

	idis := make([]int32, 0)
	for _, id := range ids {
		i, _ := strconv.Atoi(id)
		idis = append(idis, int32(i))
	}

	m.queueSend <- packetToSend{
		msg: TL_messages_getMessagesViews{
			Peer:      peer,
			Id:        idis,
			Increment: TL_boolTrue{}},
		resp: resp,
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()
	select {
	case x := <-resp:
		if x.err != nil {
			return nil, x.err
		}
		return x.data, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}
}
