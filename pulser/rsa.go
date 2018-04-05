package pulser

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

// MyPrivateKey struct
type MyPrivateKey struct {
	PublicSha1 string
	PrivateKey rsa.PrivateKey
}

// GetRsaKey returns rsa private key
func GetRsaKey() MyPrivateKey {
	return getRsaKey()
}

func getRsaKey() MyPrivateKey {
	rootDir := getRootDir()
	pubkeyFile := path.Join(rootDir, "keys", "public.key")
	privkeyFile := path.Join(rootDir, "keys", "private.key")
	if _, err := os.Stat(pubkeyFile); os.IsNotExist(err) {
		reader := rand.Reader
		bitSize := 2048

		key, err := rsa.GenerateKey(reader, bitSize)
		if err != nil {
			log.Println(err.Error())
		}

		saveGobKey(privkeyFile, key)
		saveGobKey(pubkeyFile, key.PublicKey)
	}

	mpk := MyPrivateKey{
		PrivateKey: getPrivateKey(),
		PublicSha1: getPublicSha1(),
	}
	log.Println(mpk.PrivateKey.N)
	log.Println(mpk.PrivateKey.E)
	log.Println("publik key bytes:", strings.Repeat("*", 10))
	log.Println("rsa key hex:")
	log.Printf("%x\n", mpk.PrivateKey.D.Bytes())

	log.Println("rsa public key hex:")
	log.Printf("%x\n", mpk.PrivateKey.PublicKey.N.Bytes())
	// log.Printf("%x\n", mpk.PrivateKey.PublicKey.E.Bytes())

	return mpk
}

func getPublicSha1() string {
	rootDir := getRootDir()
	pubFile := path.Join(rootDir, "keys", "public.key")
	hasher := sha1.New()

	f, err := os.Open(pubFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func getPrivateKey() rsa.PrivateKey {
	rootDir := getRootDir()
	privkeyFile := path.Join(rootDir, "keys", "private.key")

	privatekeyfile, err := os.Open(privkeyFile)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	decoder := gob.NewDecoder(privatekeyfile)

	var privatekey rsa.PrivateKey
	err = decoder.Decode(&privatekey)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	privatekeyfile.Close()

	return privatekey

}

func saveGobKey(fileName string, key interface{}) {
	outFile, err := os.Create(fileName)
	if err != nil {
		log.Println(err.Error())
	}
	defer outFile.Close()

	encoder := gob.NewEncoder(outFile)
	err = encoder.Encode(key)
	if err != nil {
		log.Println(err.Error())
	}
}
