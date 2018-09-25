package lstree

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type objectType int

// Entry represents a single ls-tree entry
type Entry struct {
	Mode   []byte
	Type   objectType
	Object string
	Path   string
}

// Entries holds every ls-tree Entry
type Entries []Entry

// Parser holds the necessary state for parsing the ls-tree output
type Parser struct {
	reader *bufio.Reader
}

const (
	delimiter            = 0
	tree      objectType = iota + 1
	blob
	submodule
)

func (e Entries) Len() int {
	return len(e)
}

func (e Entries) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

// We need to sort in the format [*tree *blobs *submodules]
func (e Entries) Less(i, j int) bool {
	return e[i].Type < e[j].Type
}

// NewParser returns a new Parser
func NewParser(src io.Reader) *Parser {
	return &Parser{
		reader: bufio.NewReader(src),
	}
}

// NextEntry reads from git ls-tree --z --full-name command
// parses the tree entry and returns a *Entry.
func (p *Parser) NextEntry() (*Entry, error) {
	result := &Entry{}

	data, err := p.reader.ReadBytes(delimiter)
	if err != nil {
		return nil, err
	}

	// We expect each `data` to be <mode> SP <type> SP <object> TAB <path>\0.
	split := bytes.SplitN(data, []byte(" "), 3)
	objectAndFile := bytes.SplitN(split[len(split)-1], []byte(" \t"), 2)
	split = append(split[:len(split)-1], objectAndFile...)

	if len(split) != 4 {
		return nil, fmt.Errorf("error parsing %q", data)
	}

	objectType, err := toEnum(string(split[1]))
	if err != nil {
		return nil, err
	}

	result.Mode = split[0]
	result.Type = objectType
	result.Object = string(split[2])

	// We know that the last byte in 'path' will be a zero byte.
	path := split[3]
	result.Path = string(path[:len(path)-1])

	return result, nil
}

func toEnum(s string) (objectType, error) {
	switch s {
	case "tree":
		return tree, nil
	case "blob":
		return blob, nil
	case "commit":
		return submodule, nil
	default:
		return -1, fmt.Errorf("Error in objectType conversion %q", s)
	}
}
