package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
