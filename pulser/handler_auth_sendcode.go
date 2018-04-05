package pulser

import (
	"pulse/pulser/mtproto"
)

func handlerHttpAuthSendCode(cd *mtproto.CacheData) []byte {

	resultContainer := mtproto.TL_msg_container{}

	cd.SeqNo++
	res := mtproto.TL_MT_message{
		Msg_id: mtproto.GenerateMessageId(),
		Seq_no: cd.SeqNo,
		Size:   28,
		Data: mtproto.TL_auth_sentCode{
			Code_type:        mtproto.TL_auth_codeTypeSms{},
			Next_type:        mtproto.TL_auth_codeTypeCall{},
			Phone_registered: false,
			Phone_code_hash:  "123",
			Timeout:          600,
		},
	}
	resultContainer.Items = append(resultContainer.Items, res)

	pack, err := mtproto.MakePacketHttp(resultContainer, cd)
	if err != nil {
		panic(err)
	}
	return pack

}
