// Package nonce generates random alphanumeric strings of fixed length.
package nonce

import (
	"math/rand"
	"strings"
	"time"
)

func init() {
	// Do not use rand.NewSource to create a new Source. Unlike the default
	// Source used by top-level rand functions, new sources are not safe for
	// concurrent use by multiple goroutines.
	rand.Seed(time.Now().UnixNano())
}

// alphanumericals contains 62 (A-Z, a-z and 0-9, case-sensitive) alphanumeric
// characters in the POSIX/C locale. The charset is ordered by the Base 64
// alphabet as defined in RFC 4648 instead of their ASCII character values:
// https://datatracker.ietf.org/doc/html/rfc4648#section-4
const alphanumericals = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

const (
	charIdxBits = 6                  // 6 bits to represent a char index.
	charIdxMask = 1<<charIdxBits - 1 // All 1-bits, as many as charIdxBits.
	charIdxMax  = 63 / charIdxBits   // # of char indices fitting in 63 bits.
)

// Generate a random alphanumeric string of the given length.
// The generate function is safe for concurrent use by multiple goroutines.
// The original algorithm is taken from: https://stackoverflow.com/a/31832326
func Generate(n int) string {
	var sb strings.Builder
	sb.Grow(n)
	for i, cache, remain := n-1, rand.Int63(), charIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), charIdxMax
		}
		if idx := int(cache & charIdxMask); idx < len(alphanumericals) {
			sb.WriteByte(alphanumericals[idx])
			i--
		}
		cache >>= charIdxBits
		remain--
	}
	return sb.String()
}
