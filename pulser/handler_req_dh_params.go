package pulser

import (
	"crypto/aes"
	"crypto/sha1"
	"errors"
	"log"
	"math/big"
	mathrand "math/rand"
	"net"
	"pulse/pulser/mtproto"
	"time"

	"github.com/davecgh/go-spew/spew"
)

var (
	dh2048_p = []byte{
		0xc7, 0x1c, 0xae, 0xb9, 0xc6, 0xb1, 0xc9, 0x04, 0x8e, 0x6c, 0x52, 0x2f,
		0x70, 0xf1, 0x3f, 0x73, 0x98, 0x0d, 0x40, 0x23, 0x8e, 0x3e, 0x21, 0xc1,
		0x49, 0x34, 0xd0, 0x37, 0x56, 0x3d, 0x93, 0x0f, 0x48, 0x19, 0x8a, 0x0a,
		0xa7, 0xc1, 0x40, 0x58, 0x22, 0x94, 0x93, 0xd2, 0x25, 0x30, 0xf4, 0xdb,
		0xfa, 0x33, 0x6f, 0x6e, 0x0a, 0xc9, 0x25, 0x13, 0x95, 0x43, 0xae, 0xd4,
		0x4c, 0xce, 0x7c, 0x37, 0x20, 0xfd, 0x51, 0xf6, 0x94, 0x58, 0x70, 0x5a,
		0xc6, 0x8c, 0xd4, 0xfe, 0x6b, 0x6b, 0x13, 0xab, 0xdc, 0x97, 0x46, 0x51,
		0x29, 0x69, 0x32, 0x84, 0x54, 0xf1, 0x8f, 0xaf, 0x8c, 0x59, 0x5f, 0x64,
		0x24, 0x77, 0xfe, 0x96, 0xbb, 0x2a, 0x94, 0x1d, 0x5b, 0xcd, 0x1d, 0x4a,
		0xc8, 0xcc, 0x49, 0x88, 0x07, 0x08, 0xfa, 0x9b, 0x37, 0x8e, 0x3c, 0x4f,
		0x3a, 0x90, 0x60, 0xbe, 0xe6, 0x7c, 0xf9, 0xa4, 0xa4, 0xa6, 0x95, 0x81,
		0x10, 0x51, 0x90, 0x7e, 0x16, 0x27, 0x53, 0xb5, 0x6b, 0x0f, 0x6b, 0x41,
		0x0d, 0xba, 0x74, 0xd8, 0xa8, 0x4b, 0x2a, 0x14, 0xb3, 0x14, 0x4e, 0x0e,
		0xf1, 0x28, 0x47, 0x54, 0xfd, 0x17, 0xed, 0x95, 0x0d, 0x59, 0x65, 0xb4,
		0xb9, 0xdd, 0x46, 0x58, 0x2d, 0xb1, 0x17, 0x8d, 0x16, 0x9c, 0x6b, 0xc4,
		0x65, 0xb0, 0xd6, 0xff, 0x9c, 0xa3, 0x92, 0x8f, 0xef, 0x5b, 0x9a, 0xe4,
		0xe4, 0x18, 0xfc, 0x15, 0xe8, 0x3e, 0xbe, 0xa0, 0xf8, 0x7f, 0xa9, 0xff,
		0x5e, 0xed, 0x70, 0x05, 0x0d, 0xed, 0x28, 0x49, 0xf4, 0x7b, 0xf9, 0x59,
		0xd9, 0x56, 0x85, 0x0c, 0xe9, 0x29, 0x85, 0x1f, 0x0d, 0x81, 0x15, 0xf6,
		0x35, 0xb1, 0x05, 0xee, 0x2e, 0x4e, 0x15, 0xd0, 0x4b, 0x24, 0x54, 0xbf,
		0x6f, 0x4f, 0xad, 0xf0, 0x34, 0xb1, 0x04, 0x03, 0x11, 0x9c, 0xd8, 0xe3,
		0xb9, 0x2f, 0xcc, 0x5b,
	}

	dh2048_g = []byte{0x02}
)

func handlerReqDHParams(data interface{}, conn net.Conn, cd *cacheData) {
	log.Println("handler req dh params")
	rMsg := data.(mtproto.TL_req_DH_params)
	spew.Dump(rMsg)
	decMsg := doRSAdecrypt(rMsg.Encdata)
	spew.Dump(decMsg)

	// TODO: check sha1 of inner data
	decBuf := mtproto.NewDecodeBuf(decMsg[20:])
	decObj := decBuf.Object()

	decData := decObj.(mtproto.TL_p_q_inner_data)
	// spew.Dump(decObj)

	newNonce := decData.New_nonce
	// dh2048p := dh2048_p
	// dh2048g := dh2048_g
	bigIntDH2048P := new(big.Int).SetBytes(dh2048_p)
	bigIntDH2048G := new(big.Int).SetBytes(dh2048_g)

	bigIntA := new(big.Int).SetBytes(generateNonce(256))

	g_a := new(big.Int)
	g_a.Exp(bigIntDH2048G, bigIntA, bigIntDH2048P)

	gs := []int{3, 4, 7}

	ed := mtproto.TL_server_DH_inner_data{
		Nonce:        cd.Nonce,
		Server_nonce: cd.ServerNonce,
		G:            int32(gs[mathrand.Intn(3)]),
		Dh_prime:     new(big.Int).SetBytes(dh2048_p),
		G_a:          g_a,
		Server_time:  int32(time.Now().Unix()),
	}

	innerP, err := mtproto.MakePacket(ed)
	if err != nil {
		panic(err)
	}
	log.Println("inner p TL_server_DH_inner_data")
	// spew.Dump(innerP)

	tmp_aes_key_and_iv := make([]byte, 64)
	sha1_a := sha1.Sum(append(newNonce, cd.ServerNonce...))
	sha1_b := sha1.Sum(append(cd.ServerNonce, newNonce...))
	sha1_c := sha1.Sum(append(newNonce, newNonce...))
	copy(tmp_aes_key_and_iv, sha1_a[:])
	copy(tmp_aes_key_and_iv[20:], sha1_b[:])
	copy(tmp_aes_key_and_iv[40:], sha1_c[:])
	copy(tmp_aes_key_and_iv[60:], newNonce[:4])

	tmpLen := 20 + len(innerP)
	if tmpLen%16 > 0 {
		tmpLen = (tmpLen/16 + 1) * 16
	} else {
		tmpLen = 20 + len(innerP)
	}

	tmp_encrypted_answer := make([]byte, tmpLen)
	sha1_tmp := sha1.Sum(innerP)
	copy(tmp_encrypted_answer, sha1_tmp[:])
	copy(tmp_encrypted_answer[20:], innerP)

	e := NewAES256IGECryptor(tmp_aes_key_and_iv[:32], tmp_aes_key_and_iv[32:64])
	tmp_encrypted_answer, _ = e.Encrypt(tmp_encrypted_answer)

	// tmp_encrypted_answer, err = doAES256IGEencrypt(tmp_encrypted_answer, tmp_aes_key_and_iv[:32], tmp_aes_key_and_iv[32:64])

	if err != nil {
		log.Println(err.Error())
	}

	resDH := mtproto.TL_server_DH_params_ok{
		Nonce:            cd.Nonce,
		Server_nonce:     cd.ServerNonce,
		Encrypted_answer: tmp_encrypted_answer,
	}
	log.Println("nonces:", cd.Nonce, cd.ServerNonce, rMsg.Nonce, newNonce)
	// spew.Dump(tmp_encrypted_answer)

	pack, err := mtproto.MakePacket(resDH)
	if err != nil {
		panic(err)
	}

	_, err = conn.Write(pack)

	log.Println("sended packet")
	// spew.Dump(pack)
	if err != nil {
		panic(err)
	}
}

func doAES256IGEencrypt(data, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data) < aes.BlockSize {
		return nil, errors.New("AES256IGE: Data too small to encrypt")
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, errors.New("AES256IGE: Data not divisible by block Size")
	}

	t := make([]byte, aes.BlockSize)
	x := make([]byte, aes.BlockSize)
	y := make([]byte, aes.BlockSize)
	copy(x, iv[:aes.BlockSize])
	copy(y, iv[aes.BlockSize:])
	encrypted := make([]byte, len(data))

	i := 0
	for i < len(data) {
		xor(x, data[i:i+aes.BlockSize])
		block.Encrypt(t, x)
		xor(t, y)
		x, y = t, data[i:i+aes.BlockSize]
		copy(encrypted[i:], t)
		i += aes.BlockSize
	}

	return encrypted, nil
}

type AES256IGECryptor struct {
	aesKey []byte
	aesIV  []byte
}

func NewAES256IGECryptor(aesKey, aesIV []byte) *AES256IGECryptor {
	// TODO(@benqi): Check aesKey and aesIV valid.
	return &AES256IGECryptor{
		aesKey: aesKey,
		aesIV:  aesIV,
	}
}

// data长度必须是aes.BlockSize(16)的倍数，如果不是请调用者补齐
func (c *AES256IGECryptor) Encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return nil, err
	}
	if len(data) < aes.BlockSize {
		return nil, errors.New("AES256IGE: data too small to encrypt")
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, errors.New("AES256IGE: data not divisible by block size")
	}

	t := make([]byte, aes.BlockSize)
	x := make([]byte, aes.BlockSize)
	y := make([]byte, aes.BlockSize)
	copy(x, c.aesIV[:aes.BlockSize])
	copy(y, c.aesIV[aes.BlockSize:])
	encrypted := make([]byte, len(data))

	i := 0
	for i < len(data) {
		xor(x, data[i:i+aes.BlockSize])
		block.Encrypt(t, x)
		xor(t, y)
		x, y = t, data[i:i+aes.BlockSize]
		copy(encrypted[i:], t)
		i += aes.BlockSize
	}

	return encrypted, nil
}

func (c *AES256IGECryptor) Decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return nil, err
	}
	if len(data) < aes.BlockSize {
		return nil, errors.New("AES256IGE: data too small to decrypt")
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, errors.New("AES256IGE: data not divisible by block size")
	}

	t := make([]byte, aes.BlockSize)
	x := make([]byte, aes.BlockSize)
	y := make([]byte, aes.BlockSize)
	copy(x, c.aesIV[:aes.BlockSize])
	copy(y, c.aesIV[aes.BlockSize:])
	decrypted := make([]byte, len(data))

	i := 0
	for i < len(data) {
		xor(y, data[i:i+aes.BlockSize])
		block.Decrypt(t, y)
		xor(t, x)
		y, x = t, data[i:i+aes.BlockSize]
		copy(decrypted[i:], t)
		i += aes.BlockSize
	}

	return decrypted, nil
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
