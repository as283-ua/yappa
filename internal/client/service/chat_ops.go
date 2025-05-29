package service

import "crypto/sha256"

const MAX_RATCHET_CYCLE = 20

func Ratchet(v []byte) []byte {
	h := sha256.New()
	h.Write(v)
	return h.Sum(nil)
}
