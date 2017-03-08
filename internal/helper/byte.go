package helper

import (
	"bytes"
)

// ByteSliceHasAnyPrefix tests whether the byte slice s begins with any of the prefixes.
func ByteSliceHasAnyPrefix(s []byte, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if bytes.HasPrefix(s, []byte(prefix)) {
			return true
		}
	}

	return false
}

// IsNumber tests whether the byte slice s contains only digits or not
func IsNumber(s []byte) bool {
	for i := range s {
		if !bytes.Contains([]byte("1234567890"), s[i:i+1]) {
			return false
		}
	}
	return true
}
