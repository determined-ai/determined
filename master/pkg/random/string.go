package random

import (
	"crypto/rand"
)

const (
	// DefaultChars is the default character set used for generating random strings.
	DefaultChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// String generates a random string of length n, using the characters in the charset string
// if provided, or using the default charset if not.
func String(n int, charset ...string) string {
	var chars string
	if len(charset) == 0 {
		chars = DefaultChars
	} else {
		chars = charset[0]
	}

	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	for i, b := range bytes {
		bytes[i] = chars[b%byte(len(chars))]
	}
	return string(bytes)
}
