package main

import (
	"log"
	"pulse/pulser/mtproto"

	sha1lib "crypto/sha1"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	obj := mtproto.GenerateNonce(256)
	z := mtproto.GenerateNonce(256)
	msgKey := sha1(z)[4:20]
	authKey := mtproto.GenerateNonce(256)

	// encryption
	aesKey, aesIV := mtproto.GenerateAES(msgKey, authKey, true)

	r := len(z) + ((16 - (len(obj) % 16)) & 15)
	log.Println("r:", r)

	y := make([]byte, len(z)+((16-(len(obj)%16))&15))
	copy(y, z)
	log.Println("z---------------------------")
	spew.Dump(z)

	encryptedData, err := mtproto.DoAES256IGEencrypt(y, aesKey, aesIV)
	if err != nil {
		panic(err)
	}
	log.Println("encrypted data-------------------")
	spew.Dump(encryptedData)

	// decryption

	aesKey, aesIV = mtproto.GenerateAES(msgKey, authKey, true)

	decryptedData, err := mtproto.DoAES256IGEdecrypt(encryptedData, aesKey, aesIV)
	if err != nil {
		panic(err)
	}
	log.Println("decryptedData data-------------------")
	spew.Dump(decryptedData)
}

func sha1(data []byte) []byte {
	r := sha1lib.Sum(data)
	return r[:]
}
