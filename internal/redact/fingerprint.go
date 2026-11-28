package redact

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Fingerprint returns the first eight hex digits of SHA-256(raw). The full URL
// is hashed so two keys on the same host stay distinct in output without
// printing either URL or token.
func Fingerprint(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:4])
}
