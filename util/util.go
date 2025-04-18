package util

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"time"
)

func RemovePort(s string) string {
	return strings.Split(s, ":")[0]
}

func RandomUint64() uint64 {
	b := make([]byte, 8)
	rand.Read(b)

	return binary.BigEndian.Uint64(b)
}

func Uint64ToBigEndian(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}

func Itob(n uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, n)
	return b
}

// returns a random float between 0 and 1
func RandomFloat() float32 {
	b := make([]byte, 4)
	rand.Read(b)

	return float32(binary.LittleEndian.Uint32(b)) / 0xffffffff
}

func Time() uint64 {
	return uint64(time.Now().Unix())
}

func AssertHex(h string) []byte {
	data, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	return data
}
