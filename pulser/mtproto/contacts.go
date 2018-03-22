package mtproto

import (
	"errors"
	"log"
	"time"
)

func (m *MTProto) ContactsGetContacts(hash string) (*TL, error) {
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_contacts_getContacts{
			Hash: hash,
		},
		resp: resp,
	}
	x := <-resp

	if x.err != nil {
		return nil, x.err
	}

	return &x.data, nil
}

func (m *MTProto) ContactsGetTopPeers(correspondents, botsPM, botsInline, groups, channels bool, offset, limit, hash int32) (*TL, error) {
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_contacts_getTopPeers{
			Correspondents: correspondents,
			Bots_pm:        botsPM,
			Bots_inline:    botsInline,
			Groups:         groups,
			Channels:       channels,
			Offset:         offset,
			Limit:          limit,
			Hash:           hash,
		},
		resp: resp,
	}
	x := <-resp
	if x.err != nil {
		return nil, x.err
	}

	switch x.data.(type) {
	case TL_contacts_topPeersNotModified:
	case TL_contacts_topPeers:
	default:
		return nil, errors.New("MTProto::ContactsGetTopPeers error: Unknown type")
	}

	return &x.data, nil
}

func (m *MTProto) ImportContact(contacts []TL) (*TL, error) {
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_contacts_importContacts{
			Contacts: contacts,
			Replace:  TL_boolFalse{},
		},
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
		return &x.data, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}
}
