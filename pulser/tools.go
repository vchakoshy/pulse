package pulser

import (
	"crypto/rand"
	"crypto/sha1"
	"log"
	"path"
	"runtime"
)

func getRootDir() string {
	_, filename, _, _ := runtime.Caller(0)
	rootDir := path.Dir(path.Dir(filename))
	log.Println("root dir:", rootDir)
	return rootDir
}

func mySha1(data []byte) []byte {
	r := sha1.Sum(data)
	return r[:]
}

func generateNonce(size int) []byte {
	b := make([]byte, size)
	_, _ = rand.Read(b)
	return b
}
