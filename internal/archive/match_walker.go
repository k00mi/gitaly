package archive

import (
	"os"
	"path/filepath"
	"regexp"
)

// Walk a directory tree, only calling the wrapped WalkFunc if the path matches
type matchWalker struct {
	wrapped  filepath.WalkFunc
	patterns []*regexp.Regexp
}

func (m matchWalker) Walk(path string, info os.FileInfo, err error) error {
	for _, pattern := range m.patterns {
		if pattern.MatchString(path) {
			return m.wrapped(path, info, err)
		}
	}

	return nil
}
