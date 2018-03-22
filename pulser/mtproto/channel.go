package mtproto

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func (m *MTProto) GetChannelsUnreadMsg(excludePinned bool, offsetDate, offsetId int32, offsetPeer TL, limit int32) (channels []ChannelInfo, err error) {
	var x response
	chat_idx := 0
	channels = make([]ChannelInfo, 0)
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

	//Set timeout for wating response
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(20 * time.Second)
		timeout <- true
	}()
	select {
	case r := <-resp:
		if r.err != nil {
			err = r.err
			return
		}
		x = r
	case <-timeout:
		log.Println("time out on response")
		err = ErrTelegramTimeOut
		return
	}

	//Parse response
	switch x.data.(type) {
	case TL_messages_dialogsSlice:
		list, ok := x.data.(TL_messages_dialogsSlice)

		if !ok {
			err = errors.New("error1")
			return
		}
		for _, v := range list.Dialogs {
			switch v.(type) {
			case TL_dialog:
				v := v.(TL_dialog)

				switch v.Peer.(type) {
				case TL_peerChat:
					chat_idx = chat_idx + 1
					continue
				case TL_peerChannel:

					if v.Unread_count == 0 {
						chat_idx = chat_idx + 1
						continue
					}

					if !list.Chats[chat_idx].(TL_channel).Megagroup {
						ch := ChannelInfo{}
						ch.ID = v.Peer.(TL_peerChannel).Channel_id
						ch.Hash = list.Chats[chat_idx].(TL_channel).Access_hash
						ch.Pts = v.Pts
						ch.UnreadCount = v.Unread_count
						ch.TopMessage = v.Top_message
						ch.Title = list.Chats[chat_idx].(TL_channel).Title
						ch.Username = list.Chats[chat_idx].(TL_channel).Username

						channels = append(channels, ch)
						chat_idx = chat_idx + 1
					} else {
						chat_idx = chat_idx + 1
						continue
					}
				}
			}
		}
	case TL_messages_dialogs:
		list, ok := x.data.(TL_messages_dialogs)

		if !ok {
			err = errors.New("error2")
			return
		}
		for _, v := range list.Dialogs {
			switch v.(type) {
			case TL_dialog:
				v := v.(TL_dialog)

				switch v.Peer.(type) {
				case TL_peerChat:
					chat_idx = chat_idx + 1
					continue
				case TL_peerChannel:

					if v.Unread_count == 0 {
						chat_idx = chat_idx + 1
						continue
					}

					if !list.Chats[chat_idx].(TL_channel).Megagroup {
						ch := ChannelInfo{}
						ch.ID = v.Peer.(TL_peerChannel).Channel_id
						ch.Hash = list.Chats[chat_idx].(TL_channel).Access_hash
						ch.Pts = v.Pts
						ch.UnreadCount = v.Unread_count
						ch.TopMessage = v.Top_message
						ch.Title = list.Chats[chat_idx].(TL_channel).Title
						ch.Username = list.Chats[chat_idx].(TL_channel).Username

						channels = append(channels, ch)
						chat_idx = chat_idx + 1
					} else {
						chat_idx = chat_idx + 1
						continue
					}
				}
			}
		}
	default:
		spew.Dump(x.data)
	}
	return
}

func (m *MTProto) GetChannels(str string) error {
	log.Println("Get channels")
	resp := make(chan response, 1)
	channels := strings.Split(os.Args[2], "&")
	tl := []TL{}
	for _, channel := range channels {
		ch := strings.Split(channel, ",")
		chID, _ := strconv.Atoi(ch[0])
		hash, _ := strconv.ParseInt(ch[1], 10, 64)
		tl = append(tl, TL_inputChannel{int32(chID), hash})
	}

	m.queueSend <- packetToSend{TL_channels_getChannels{tl}, resp}
	x := <-resp
	if x.err != nil {
		return x.err
	}
	list, ok := x.data.(TL_messages_chats)
	if !ok {
		return fmt.Errorf("RPC: %#v", x)
	}
	fmt.Printf("%#v", list)

	return nil
}

func (m *MTProto) GetParticipants(channelID int32, accessHash int64) (count int32, err error) {
	var x response
	resp := make(chan response, 1)

	m.queueSend <- packetToSend{TL_channels_getFullChannel{TL_inputChannel{channelID, accessHash}}, resp}

	//Set timeout for wating response
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()
	select {
	case r := <-resp:
		if r.err != nil {
			err = r.err
			return
		}
		x = r
	case <-timeout:
		log.Println("time out on response")
		err = ErrTelegramTimeOut
		return
	}

	//Parse response
	switch x.data.(type) {
	case TL_messages_chatFull:
		count = x.data.(TL_messages_chatFull).Full_chat.(TL_channelFull).ParticipantsCount
	default:
		count = 0
	}
	return
}

func (m *MTProto) GetFullChannel(channelID int32, accessHash int64) (TL_channelFull, error) {
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{TL_channels_getFullChannel{TL_inputChannel{channelID, accessHash}}, resp}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()
	select {
	case x := <-resp:
		if x.err != nil {
			return TL_channelFull{}, x.err
		}
		return x.data.(TL_messages_chatFull).Full_chat.(TL_channelFull), nil
	case <-timeout:
		log.Println("time out on response")
		return TL_channelFull{}, ErrTelegramTimeOut
	}
}

func (m *MTProto) GetChannelFromGlobalSearch(username string) (TL_channel, error) {
	var (
		channelId int32
		channel   TL_channel
		found     bool
		x         response
	)
	channels := make(map[int32]TL_channel)
	resp := make(chan response, 1)

	m.queueSend <- packetToSend{

		TL_messages_searchGlobal{
			Limit:       20,
			Offset_date: 0,
			Offset_id:   0,
			Offset_peer: TL_inputPeerEmpty{},
			Q:           username,
		},
		resp}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()
	select {
	case r := <-resp:
		if r.err != nil {
			return channel, r.err
		}
		x = r
	case <-timeout:
		log.Println("time out on response")
		return channel, ErrTelegramTimeOut
	}

	spew.Dump(x)

	// chats := x.data.(TL_messages_messages)
	// for _, chat := range chats.Chats {
	// 	c := chat.(TL_chat)
	// 	spew.Dump(c)
	// }

	switch x.data.(type) {
	case TL_contacts_resolvedPeer:
		peer := x.data.(TL_contacts_resolvedPeer).Peer
		switch peer.(type) {
		case TL_peerChannel:
			channelId = peer.(TL_peerChannel).Channel_id
		}
		chats := x.data.(TL_contacts_resolvedPeer).Chats
		for _, chat := range chats {
			switch chat.(type) {
			case TL_channel:
				ch := chat.(TL_channel)
				channels[ch.Id] = ch
			}
		}
	}

	channel, found = channels[channelId]
	if !found {
		err := errors.New("channel not found")
		return channel, err
	}
	return channel, nil
}

func (m *MTProto) GetChannelFromUsername(username string) (TL_channel, error) {
	var (
		channelId int32
		channel   TL_channel
		found     bool
		x         response
	)
	channels := make(map[int32]TL_channel)
	resp := make(chan response, 1)

	m.queueSend <- packetToSend{
		TL_contacts_resolveUsername{
			Username: username},
		resp}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()
	select {
	case r := <-resp:
		if r.err != nil {
			return channel, r.err
		}
		x = r
	case <-timeout:
		log.Println("time out on response")
		return channel, ErrTelegramTimeOut
	}

	switch x.data.(type) {
	case TL_contacts_resolvedPeer:
		peer := x.data.(TL_contacts_resolvedPeer).Peer
		switch peer.(type) {
		case TL_peerChannel:
			channelId = peer.(TL_peerChannel).Channel_id
		}
		chats := x.data.(TL_contacts_resolvedPeer).Chats
		for _, chat := range chats {
			switch chat.(type) {
			case TL_channel:
				ch := chat.(TL_channel)
				channels[ch.Id] = ch
			}
		}
	}

	channel, found = channels[channelId]
	if !found {
		err := errors.New("channel not found")
		return channel, err
	}
	return channel, nil
}

func (m *MTProto) GetUserFromUsername(username string) (TL_user, error) {
	var (
		user TL_user
		x    response
	)
	resp := make(chan response, 1)

	m.queueSend <- packetToSend{
		TL_contacts_resolveUsername{
			Username: username},
		resp}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()
	select {
	case r := <-resp:
		if r.err != nil {
			return user, r.err
		}
		x = r
	case <-timeout:
		log.Println("time out on response")
		return user, ErrTelegramTimeOut
	}

	switch x.data.(type) {
	case TL_contacts_resolvedPeer:
		users := x.data.(TL_contacts_resolvedPeer).Users
		for _, u := range users {
			switch u.(type) {
			case TL_user:
				user := u.(TL_user)
				if user.Username == username {
					return user, nil
				}

			}
		}
	}

	return TL_user{}, errors.New("user not found")
}
func (m *MTProto) JoinChannel(channelId int32, hash int64) (TL, error) {

	channel := TL_inputChannel{Channel_id: channelId, Access_hash: hash}
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		TL_channels_joinChannel{Channel: channel},
		resp}

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

func (m *MTProto) LeaveChannel(channelId int32, hash int64) error {

	channel := TL_inputChannel{Channel_id: channelId, Access_hash: hash}
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		TL_channels_leaveChannel{Channel: channel},
		resp}
	x := <-resp
	if x.err != nil {
		return x.err
	}
	return nil
}

func (m *MTProto) GetMessageObject(msg TL_message, channelId int32) Message {
	uniqeID := fmt.Sprintf("telegram.com/%d/%d", channelId, msg.Id)
	data := []byte(uniqeID)

	message := Message{}
	message.ID = fmt.Sprintf("%x", md5.Sum(data))
	message.Text = msg.Message
	message.Date = msg.Date
	message.ViewCount = msg.Views
	message.Media = msg.Media

	return message
}

func (m *MTProto) GetChannelObjects(channel TL_channel) ChannelInfo {
	info := ChannelInfo{}
	info.Title = channel.Title
	info.Username = channel.Username
	info.ID = channel.Id
	info.Hash = channel.Access_hash
	// info.ParticipantsCount, err = m.GetParticipants(channel.Id, channel.Access_hash)
	return info
}

func (m *MTProto) ChannelsReadHistory(channelID, maxID int32, accessHash int64) error {
	resp := make(chan response, 1)
	channel := TL_inputChannel{channelID, accessHash}
	m.queueSend <- packetToSend{TL_channels_readHistory{channel, maxID}, resp}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()
	select {
	case x := <-resp:
		if x.err != nil {
			return x.err
		}
		return nil
	case <-timeout:
		return ErrTelegramTimeOut
	}
}

func (m *MTProto) GetChannelParticipants(channelID int32, hash int64, offset, limit int32) (TL, error) {
	resp := make(chan response, 1)
	channel := TL_inputChannel{Channel_id: channelID, Access_hash: hash}
	filter := TL_channelParticipantsRecent{}
	m.queueSend <- packetToSend{
		TL_channels_getParticipants{
			Channel: channel,
			Filter:  filter,
			Offset:  offset,
			Limit:   limit},
		resp}

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
		return nil, ErrTelegramTimeOut
	}
}
