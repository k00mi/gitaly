package git

import (
	"fmt"
	"regexp"
)

var commitIDRegex = regexp.MustCompile(`\A[0-9a-f]{40}\z`)

// ValidateCommitID checks if id could be a Git commit ID, syntactically.
func ValidateCommitID(id string) error {
	if commitIDRegex.MatchString(id) {
		return nil
	}

	return fmt.Errorf("invalid commit ID: %q", id)
}
