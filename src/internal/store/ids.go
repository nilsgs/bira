package store

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID generates an 8-character alphanumeric identifier using crypto/rand.
func NewID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
