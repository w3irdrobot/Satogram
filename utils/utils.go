package utils

import (
	crypto_rand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
)

func RandomInt32() int32 {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		panic("cannot seed math/rand package with cryptographically secure random number generator")
	}
	rand.New(rand.NewSource(int64(binary.LittleEndian.Uint64(b[:]))))
	// Generate a random 32-bit integer
	randomInt := rand.Int31()

	return randomInt
}

func GenerateSHA256Hash(keyBytes []byte) []byte {
	var sha256Hasher = sha256.New()
	sha256Hasher.Write(keyBytes)

	return sha256Hasher.Sum(nil)
}
