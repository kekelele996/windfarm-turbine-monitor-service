package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Signer produces and verifies HMAC signatures for payloads.
type Signer struct {
	secret []byte
}

func NewSigner(secret string) *Signer { return &Signer{secret: []byte(secret)} }

// Sign returns the hex HMAC-SHA256 of payload.
func (s *Signer) Sign(payload []byte) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks a hex signature against the payload.
func (s *Signer) Verify(payload []byte, signature string) (bool, error) {
	want, err := hex.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}
	got := hmac.New(sha256.New, s.secret)
	_, _ = got.Write(payload)
	return hmac.Equal(got.Sum(nil), want), nil
}
