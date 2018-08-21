package helper

import (
	"os"
	"strings"
)

// ContainsPathTraversal checks if the path contains any traversal
func ContainsPathTraversal(path string) bool {
	// Disallow directory traversal for security
	separator := string(os.PathSeparator)
	return strings.HasPrefix(path, ".."+separator) ||
		strings.Contains(path, separator+".."+separator) ||
		strings.HasSuffix(path, separator+"..")
}
