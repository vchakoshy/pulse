package pulser

import (
	"net"
	"pulse/pulser/mtproto"
)

func handlerinvokeWithLayer(data interface{}, conn net.Conn, cd *mtproto.CacheData) {

	// (mtproto.TL_msg_container) {
	// 	Items: ([]mtproto.TL_MT_message) (len=2 cap=2) {
	// 	 (mtproto.TL_MT_message) {
	// 	  Msg_id: (int64) 6537381686119440385,
	// 	  Seq_no: (int32) 1,
	// 	  Size: (int32) 28,
	// 	  Data: (mtproto.TL_new_session_created) {
	// 	   First_msg_id: (int64) 6537381683531272692,
	// 	   Unique_id: (int64) 8297134183777339944,
	// 	   Server_salt: ([]uint8) (len=8 cap=8) {
	// 		00000000  38 50 79 34 26 47 fd ca                           |8Py4&G..|
	// 	   }
	// 	  }
	// 	 },
	// 	 (mtproto.TL_MT_message) {
	// 	  Msg_id: (int64) 6537381686119489537,
	// 	  Seq_no: (int32) 2,
	// 	  Size: (int32) 20,
	// 	  Data: (mtproto.TL_msgs_ack) {
	// 	   MsgIds: ([]int64) (len=1 cap=1) {
	// 		(int64) 6537381683531272692
	// 	   }
	// 	  }
	// 	 }
	// 	}
	//    }
	cd.SeqNo++
	nsMtMsg := mtproto.TL_MT_message{
		Msg_id: mtproto.GenerateMessageId(),
		Seq_no: cd.SeqNo,
		Size:   28,
		Data: mtproto.TL_new_session_created{
			First_msg_id: cd.MsgID,
			Unique_id:    8297134183777339944,
			Server_salt:  cd.ServerSalt,
		},
	}
	cd.SeqNo++
	msgAck := mtproto.TL_MT_message{
		Msg_id: mtproto.GenerateMessageId(),
		Seq_no: cd.SeqNo,
		Size:   20,
		Data: mtproto.TL_msgs_ack{
			MsgIds: []int64{cd.MsgID},
		},
	}
	cd.SeqNo++
	tlmsgCon := mtproto.TL_msg_container{}
	tlmsgCon.Items = append(tlmsgCon.Items, nsMtMsg)
	tlmsgCon.Items = append(tlmsgCon.Items, msgAck)

	// d := data.(mtproto.TL_invokeWithLayer)
	// spew.Dump(d)
	// log.Println(cd.MsgID)

	pack, err := mtproto.MakingPacket(tlmsgCon, cd)
	if err != nil {
		panic(err)
	}

	_, err = conn.Write(pack)
	if err != nil {
		panic(err)
	}

	// log.Println("invoke with layer")
	// tlConfL := mtproto.TL_config{
	// 	Flags:              0,
	// 	Phonecalls_enabled: false,
	// 	Date:               int32(time.Now().Unix()),
	// 	Expires:            int32(time.Now().Unix()) + 86400,
	// 	Test_mode:          mtproto.TL_boolFalse{},
	// 	This_dc:            2,
	// 	Dc_options: []mtproto.TL{
	// 		mtproto.TL_dcOption{
	// 			Flags:      0,
	// 			Ipv6:       false,
	// 			Media_only: false,
	// 			Tcpo_only:  false,
	// 			Id:         1,
	// 			Ip_address: "127.0.0.1",
	// 			Port:       443}},
	// 	Chat_size_max:            200,
	// 	Megagroup_size_max:       100000,
	// 	Forwarded_count_max:      100,
	// 	Online_update_period_ms:  120000,
	// 	Offline_blur_timeout_ms:  5000,
	// 	Offline_idle_timeout_ms:  30000,
	// 	Online_cloud_timeout_ms:  300000,
	// 	Notify_cloud_delay_ms:    30000,
	// 	Notify_default_delay_ms:  1500,
	// 	Chat_big_size:            10,
	// 	Push_chat_period_ms:      60000,
	// 	Push_chat_limit:          2,
	// 	Saved_gifs_limit:         200,
	// 	Edit_time_limit:          172800,
	// 	Rating_e_decay:           2419200,
	// 	Stickers_recent_limit:    30,
	// 	Tmp_sessions:             0,
	// 	Pinned_dialogs_count_max: 5,
	// 	Call_receive_timeout_ms:  20000,
	// 	Call_ring_timeout_ms:     90000,
	// 	Call_connect_timeout_ms:  30000,
	// 	Call_packet_timeout_ms:   10000,
	// 	Me_url_prefix:            "https://t.me/",
	// 	Disabled_features:        []mtproto.TL{},
	// }

	// pack, err := mtproto.MakePacket(tlConfL)
	// if err != nil {
	// 	panic(err)
	// }

}
