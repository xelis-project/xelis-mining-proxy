package xelisutil

import (
	"github.com/zeebo/blake3"
)

func FastHash(d []byte) [32]byte {
	return blake3.Sum256(d)
}
