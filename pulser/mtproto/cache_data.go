package mtproto

import "math/big"

type CacheData struct {
	Nonce       []byte
	ServerNonce []byte
	NewNonce    []byte
	TmpAESKey   []byte
	TmpAESIV    []byte
	AuthKeyID   int64
	AuthKey     []byte
	AuthKeyHash []byte
	GA          *big.Int
	G           int32
	A           *big.Int
	MsgID       int64
	SeqNo       int32
	ServerSalt  []byte
	SessionID   int64
	Encrypted   bool
	ReqMsgID    int64
}
