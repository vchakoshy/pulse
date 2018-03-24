package pulser

import (
	"crypto/sha1"
	"log"
	"math/big"
	"net"
	"pulse/pulser/mtproto"
)

func handlerSetClientDHParams(data interface{}, conn net.Conn, cd *mtproto.CacheData) {
	log.Println("handlerSetClientDHParams")
	rMsg := data.(mtproto.TL_set_client_DH_params)

	bEncryptedData := rMsg.Encdata

	tmp_aes_key_and_iv := make([]byte, 64)
	sha1_a := sha1.Sum(append(cd.NewNonce, cd.ServerNonce...))
	sha1_b := sha1.Sum(append(cd.ServerNonce, cd.NewNonce...))
	sha1_c := sha1.Sum(append(cd.NewNonce, cd.NewNonce...))
	copy(tmp_aes_key_and_iv, sha1_a[:])
	copy(tmp_aes_key_and_iv[20:], sha1_b[:])
	copy(tmp_aes_key_and_iv[40:], sha1_c[:])
	copy(tmp_aes_key_and_iv[60:], cd.NewNonce[:4])

	decryptedData, err := doAES256IGEdecrypt(bEncryptedData, tmp_aes_key_and_iv[:32], tmp_aes_key_and_iv[32:64])

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

	t4 := make([]byte, 32+1+8)
	copy(t4[0:], cd.NewNonce)
	t4[32] = 1
	copy(t4[33:], mySha1(authKey.Bytes())[0:8])

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

	_, err = conn.Write(pack)

	if err != nil {
		panic(err)
	}

}
