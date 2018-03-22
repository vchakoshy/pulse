package mtproto

import (
	"log"
	"time"
)

func (m *MTProto) UpdatesGetState() (TL, error) {
	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg:  TL_updates_getState{},
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

func (m *MTProto) UpdatesGetDifference(pts, ptsTotalLimit, date, qts int32) (*TL, error) {
	resp := make(chan response, 1)

	packTs := packetToSend{
		msg: TL_updates_getDifference{
			Pts:             pts,
			Pts_total_limit: ptsTotalLimit,
			Date:            date,
			Qts:             qts,
		},
		resp: resp,
	}
	m.queueSend <- packTs

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

func (m *MTProto) UpdatesGetChannelDifference(force bool, channel, filter TL, pts, limit int32) (TL, error) {

	resp := make(chan response, 1)
	m.queueSend <- packetToSend{
		msg: TL_updates_getChannelDifference{
			// Force:   force,
			Channel: channel,
			Filter:  filter,
			Pts:     pts,
			Limit:   limit,
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
		return x.data, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
