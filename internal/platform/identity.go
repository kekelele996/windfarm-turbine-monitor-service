package platform

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewID returns a 16-byte random identifier as a hex string.
func NewID(prefix string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

// NewSequenceID builds a short, time-ordered identifier for a given subject.
func NewSequenceID(prefix string, n uint64) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), n)
}
