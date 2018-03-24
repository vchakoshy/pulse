package pulser

import (
	"crypto/sha1"
	"log"
	"math/big"
	mathrand "math/rand"
	"net"
	"pulse/pulser/mtproto"
	"time"
)

func handlerReqDHParams(data interface{}, conn net.Conn, cd *mtproto.CacheData) {
	rMsg := data.(mtproto.TL_req_DH_params)
	decMsg := doRSAdecrypt(rMsg.Encdata)

	// TODO: check sha1 of inner data
	decBuf := mtproto.NewDecodeBuf(decMsg[20:])
	decObj := decBuf.Object()

	decData := decObj.(mtproto.TL_p_q_inner_data)

	newNonce := decData.New_nonce
	cd.NewNonce = newNonce

	bigIntDH2048P := new(big.Int).SetBytes(dh2048_p)

	cd.A = new(big.Int).SetBytes(generateNonce(256))

	gs := []int{3, 4, 7}
	cd.G = int32(gs[mathrand.Intn(3)])

	cd.GA = new(big.Int).Exp(big.NewInt(int64(cd.G)), cd.A, bigIntDH2048P)

	ed := mtproto.TL_server_DH_inner_data{
		Nonce:        cd.Nonce,
		Server_nonce: cd.ServerNonce,
		G:            cd.G,
		Dh_prime:     new(big.Int).SetBytes(dh2048_p),
		G_a:          cd.GA,
		Server_time:  int32(time.Now().Unix()),
	}

	innerP := mtproto.EncodeTL(ed)

	tmpAesKeyIv := make([]byte, 64)
	sha1A := sha1.Sum(append(newNonce, cd.ServerNonce...))
	sha1B := sha1.Sum(append(cd.ServerNonce, newNonce...))
	sha1C := sha1.Sum(append(newNonce, newNonce...))
	copy(tmpAesKeyIv, sha1A[:])
	copy(tmpAesKeyIv[20:], sha1B[:])
	copy(tmpAesKeyIv[40:], sha1C[:])
	copy(tmpAesKeyIv[60:], newNonce[:4])

	tmpLen := 20 + len(innerP)
	if tmpLen%16 > 0 {
		tmpLen = (tmpLen/16 + 1) * 16
	} else {
		tmpLen = 20 + len(innerP)
	}

	tmpEncryptedAnswer := make([]byte, tmpLen)
	sha1Tmp := sha1.Sum(innerP)
	copy(tmpEncryptedAnswer, sha1Tmp[:])
	copy(tmpEncryptedAnswer[20:], innerP)

	aesKey := tmpAesKeyIv[:32]
	cd.TmpAESKey = aesKey
	aesIV := tmpAesKeyIv[32:64]
	cd.TmpAESIV = aesIV

	encryptedAnswer, err := doAES256IGEencrypt(tmpEncryptedAnswer, aesKey, aesIV)
	if err != nil {
		log.Println(err.Error())
	}

	resDH := mtproto.TL_server_DH_params_ok{
		Nonce:            cd.Nonce,
		Server_nonce:     cd.ServerNonce,
		Encrypted_answer: encryptedAnswer,
	}

	pack, err := mtproto.MakePacket(resDH)
	if err != nil {
		panic(err)
	}

	_, err = conn.Write(pack)

	if err != nil {
		panic(err)
	}
}

func xor(dst, src []byte) {
	for i := range dst {
		dst[i] = dst[i] ^ src[i]
	}
}

func doRSAdecrypt(em []byte) []byte {
	z := make([]byte, 255)
	copy(z, em)

	c := new(big.Int)
	r := getPrivateKey()
	c.Exp(new(big.Int).SetBytes(em), r.D, r.N)

	res := make([]byte, 256)
	copy(res, c.Bytes())

	return res
}
