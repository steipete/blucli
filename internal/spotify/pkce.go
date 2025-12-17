package spotify

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func NewCodeVerifier() (string, error) {
	// 32 bytes -> 43 chars base64url (no padding), within RFC 7636 recommended range.
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func CodeChallengeS256(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
