package lstree

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// Entry represents a single ls-tree entry
type Entry struct {
	Mode   []byte
	Type   []byte
	Object string
	Path   string
}

// Parser holds the necessary state for parsing the ls-tree output
type Parser struct {
	reader *bufio.Reader
}

const (
	delimiter = 0
)

// NewParser returns a new Parser
func NewParser(src io.Reader) *Parser {
	return &Parser{
		reader: bufio.NewReader(src),
	}
}

// NextPath reads from git ls-tree --z --full-name command
// parses the tree entry and returns a *Entry.
func (p *Parser) NextEntry() (*Entry, error) {
	result := &Entry{}

	data, err := p.reader.ReadBytes(delimiter)
	if err != nil {
		return nil, err
	}

	// We expect each `data` to be <mode> SP <type> SP <object> TAB <path>\0.
	split := bytes.Split(data, []byte(" "))
	objectAndFile := bytes.Split(split[len(split)-1], []byte(" \t"))
	split = append(split[:len(split)-1], objectAndFile...)

	if len(split) != 4 {
		return nil, fmt.Errorf("error parsing %q", data)
	}

	result.Mode = split[0]
	result.Type = split[1]
	result.Object = string(split[2][:])

	// We know that the last byte in 'path' will be a zero byte.
	path := split[3]
	result.Path = string(path[:len(path)-1])

	return result, nil
}
