package main

import (
	"crypto/sha256"
	"encoding/hex"
)

func docID(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:8])
}

