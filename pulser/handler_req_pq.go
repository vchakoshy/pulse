package pulser

import (
	"crypto/rand"
	"encoding/binary"
	"log"
	"math/big"
	"net"
	"pulse/pulser/mtproto"
)

func handlerReqPQ(data interface{}, conn net.Conn, cd *cacheData) {
	rsaKey := getRsaKey()
	log.Println(rsaKey.PublicSha1)

	fp := make([]byte, 9)
	for i := 0; i < 8; i++ {
		fp[i] = rsaKey.PublicSha1[len(rsaKey.PublicSha1)-(8-i)]
	}

	fpEnd := binary.BigEndian.Uint64(fp)

	recData := data.(mtproto.TL_req_pq)
	serverNone := generateNonce(16)

	tlResPQ := mtproto.TL_resPQ{
		Nonce:        recData.Nonce,
		Server_nonce: serverNone,
		Pq:           calculatePq(),
		Fingerprints: []int64{int64(fpEnd)},
	}
	pack, err := mtproto.MakePacket(tlResPQ)
	if err != nil {
		panic(err)
	}

	cd.Nonce = recData.Nonce
	cd.ServerNonce = serverNone

	_, err = conn.Write(pack)
	if err != nil {
		panic(err)
	}
}

func generateNonce(size int) []byte {
	b := make([]byte, size)
	_, _ = rand.Read(b)
	return b
}

func calculatePq() *big.Int {
	var p, q *big.Int
	a := getRandomPrime()
	b := getRandomPrime()

	comparison := a.Cmp(b)
	if comparison == -1 {
		p = a
		q = b
	} else {
		p = b
		q = a
	}

	pq := new(big.Int).Mul(p, q)
	return pq
}

func getRandomPrime() *big.Int {
	rnderd := rand.Reader
	num, _ := rand.Int(rnderd, big.NewInt(999000000))
	num.Add(num, big.NewInt(1000000000))

	probablePrime := nexPrime(num)
	if probablePrime.Int64() < 2000000000 && probablePrime.Int64() > 1000000000 {
		return probablePrime
	}
	return getRandomPrime()
}

func nexPrime(start *big.Int) *big.Int {
	for i := start.Int64(); ; i++ {
		j := big.NewInt(i)
		isPrime := j.ProbablyPrime(1)
		if isPrime {
			return j
		}
	}
}
