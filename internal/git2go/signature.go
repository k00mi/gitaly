package git2go

import (
	"strings"
	"time"
)

var signatureSanitizer = strings.NewReplacer("\n", "", "<", "", ">", "")

// Signature represents a commits signature.
type Signature struct {
	// Name of the author or the committer.
	Name string
	// Email of the author or the committer.
	Email string
	// When is the time of the commit.
	When time.Time
}

// NewSignature creates a new sanitized signature.
func NewSignature(name, email string, when time.Time) Signature {
	return Signature{
		Name:  signatureSanitizer.Replace(name),
		Email: signatureSanitizer.Replace(email),
		When:  when.Truncate(time.Second),
	}
}
