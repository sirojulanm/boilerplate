package helper

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

const requestIDBytes = 8

// NewRequestID returns a random correlation ID, falling back to a timestamp if
// the system random source is unavailable.
func NewRequestID() string {
	bytes := make([]byte, requestIDBytes)
	if _, err := rand.Read(bytes); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(bytes)
}
