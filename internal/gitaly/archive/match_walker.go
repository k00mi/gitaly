package archive

import (
	"os"
	"path/filepath"
	"regexp"
)

// MatchWalker walks a directory tree, only calling the wrapped WalkFunc if the path matches
type MatchWalker struct {
	wrapped  filepath.WalkFunc
	patterns []*regexp.Regexp
}

// Walk walks the tree, filtering on regexp patterns
func (m MatchWalker) Walk(path string, info os.FileInfo, err error) error {
	for _, pattern := range m.patterns {
		if pattern.MatchString(path) {
			return m.wrapped(path, info, err)
		}
	}

	return nil
}

// NewMatchWalker returns a new MatchWalker given a slice of patterns and a filepath.WalkFunc
func NewMatchWalker(patterns []*regexp.Regexp, walkFunc filepath.WalkFunc) *MatchWalker {
	return &MatchWalker{
		wrapped:  walkFunc,
		patterns: patterns,
	}
}
