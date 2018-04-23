package pulser

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/vchakoshy/pulse/pulser/mtproto"
)

func handlerSetClientDHParams(data interface{}, conn net.Conn, cd *mtproto.CacheData) {
	log.Println("handlerSetClientDHParams")
	rMsg := data.(mtproto.TL_set_client_DH_params)

	bEncryptedData := rMsg.Encdata

	tmpAesKeyIv := make([]byte, 64)
	sha1A := sha1.Sum(append(cd.NewNonce, cd.ServerNonce...))
	sha1B := sha1.Sum(append(cd.ServerNonce, cd.NewNonce...))
	sha1C := sha1.Sum(append(cd.NewNonce, cd.NewNonce...))
	copy(tmpAesKeyIv, sha1A[:])
	copy(tmpAesKeyIv[20:], sha1B[:])
	copy(tmpAesKeyIv[40:], sha1C[:])
	copy(tmpAesKeyIv[60:], cd.NewNonce[:4])

	decryptedData, err := doAES256IGEdecrypt(bEncryptedData, tmpAesKeyIv[:32], tmpAesKeyIv[32:64])

	if err != nil {
		log.Println(err.Error())
	}

	innerbuf := mtproto.NewDecodeBuf(decryptedData[20:])
	data = innerbuf.Object()
	if innerbuf.Error() != nil {
		log.Println("error in find object", innerbuf.Error().Error())
	}

	dhi, ok := data.(mtproto.TL_client_DH_inner_data)
	if !ok {
		log.Println("set clint dh params not ok")
		return
	}

	authKey := new(big.Int).Exp(
		dhi.G_b,
		cd.A,
		new(big.Int).SetBytes(dh2048_p))

	cd.AuthKey = authKey.Bytes()

	cd.AuthKeyHash = mySha1(cd.AuthKey)[12:20]

	t4 := make([]byte, 32+1+8)
	copy(t4[0:], cd.NewNonce)
	t4[32] = 1
	copy(t4[33:], mySha1(authKey.Bytes())[0:8])

	cd.ServerSalt = make([]byte, 8)
	copy(cd.ServerSalt, cd.NewNonce[:8])
	xor(cd.ServerSalt, cd.ServerNonce[:8])

	nonceHash1 := mySha1(t4)[4:20]

	tlok := mtproto.TL_dh_gen_ok{
		New_nonce_hash1: nonceHash1,
		Nonce:           cd.Nonce,
		Server_nonce:    cd.ServerNonce,
	}

	pack, err := mtproto.MakePacket(tlok)
	if err != nil {
		panic(err)
	}

	cd.Encrypted = true

	_, err = conn.Write(pack)

	if err != nil {
		panic(err)
	}

	// TODO: store auth key

}

func HandlerHttpSetClientDHParams(data interface{}, sessions map[string]*mtproto.CacheData) []byte {
	log.Println("handlerSetClientDHParams")
	rMsg := data.(mtproto.TL_set_client_DH_params)

	cd := sessions[string(rMsg.Nonce)]

	bEncryptedData := rMsg.Encdata

	tmpAesKeyIv := make([]byte, 64)
	sha1A := sha1.Sum(append(cd.NewNonce, cd.ServerNonce...))
	sha1B := sha1.Sum(append(cd.ServerNonce, cd.NewNonce...))
	sha1C := sha1.Sum(append(cd.NewNonce, cd.NewNonce...))
	copy(tmpAesKeyIv, sha1A[:])
	copy(tmpAesKeyIv[20:], sha1B[:])
	copy(tmpAesKeyIv[40:], sha1C[:])
	copy(tmpAesKeyIv[60:], cd.NewNonce[:4])

	decryptedData, err := doAES256IGEdecrypt(bEncryptedData, tmpAesKeyIv[:32], tmpAesKeyIv[32:64])

	if err != nil {
		log.Println(err.Error())
	}

	innerbuf := mtproto.NewDecodeBuf(decryptedData[20:])
	data = innerbuf.Object()
	if innerbuf.Error() != nil {
		log.Println("error in find object", innerbuf.Error().Error())
	}

	dhi, ok := data.(mtproto.TL_client_DH_inner_data)
	if !ok {
		log.Println("set clint dh params not ok")
		return nil
	}

	authKey := new(big.Int).Exp(
		dhi.G_b,
		cd.A,
		new(big.Int).SetBytes(dh2048_p))

	cd.AuthKey = authKey.Bytes()

	cd.AuthKeyHash = mySha1(cd.AuthKey)[12:20]

	t4 := make([]byte, 32+1+8)
	copy(t4[0:], cd.NewNonce)
	t4[32] = 1
	copy(t4[33:], mySha1(authKey.Bytes())[0:8])

	cd.ServerSalt = make([]byte, 8)
	copy(cd.ServerSalt, cd.NewNonce[:8])
	xor(cd.ServerSalt, cd.ServerNonce[:8])

	nonceHash1 := mySha1(t4)[4:20]

	tlok := mtproto.TL_dh_gen_ok{
		New_nonce_hash1: nonceHash1,
		Nonce:           cd.Nonce,
		Server_nonce:    cd.ServerNonce,
	}

	pack, err := mtproto.MakePacketHttp(tlok, nil)
	if err != nil {
		panic(err)
	}

	cd.Encrypted = true

	spew.Dump(authKeyHashes)

	authKeySha1 := fmt.Sprintf("%x", mySha1(cd.AuthKeyHash))

	rCon := getRedisClient()
	cdData, _ := json.Marshal(cd)
	err = rCon.Set("authkey:"+authKeySha1, cdData, time.Hour*24).Err()
	if err != nil {
		log.Println(err.Error())
	}

	authKeyHashes[authKeySha1] = cd

	return pack

	// TODO: store auth key

}
