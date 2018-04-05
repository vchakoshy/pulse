package pulser

import (
	"crypto/rand"
	"crypto/sha1"
	"path"
	"runtime"

	redis "gopkg.in/redis.v4"
)

func getRootDir() string {
	_, filename, _, _ := runtime.Caller(0)
	rootDir := path.Dir(path.Dir(filename))
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

var redisClient *redis.Client

func getRedisClient() *redis.Client {
	if redisClient != nil {
		return redisClient
	}
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return redisClient
}
