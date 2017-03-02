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
