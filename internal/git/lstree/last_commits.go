package lstree

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

// Enum for the type of object in an ls-tree entry
// can be tree, blob or commit
type ObjectType int

// Entry represents a single ls-tree entry
type Entry struct {
	Mode []byte
	Type ObjectType
	Oid  string
	Path string
}

// Entries holds every ls-tree Entry
type Entries []Entry

// Parser holds the necessary state for parsing the ls-tree output
type Parser struct {
	reader *bufio.Reader
}

const (
	tree ObjectType = iota
	blob
	submodule
)

// ErrParse is returned when the parse of an entry was unsuccessful
var ErrParse = errors.New("Failed to parse git ls-tree response")

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
	data, err := p.reader.ReadBytes(0x00)
	if err != nil {
		return nil, err
	}

	// We expect each `data` to be <mode> SP <type> SP <object> TAB <path>\0.
	split := bytes.SplitN(data, []byte(" "), 3)
	if len(split) != 3 {
		return nil, ErrParse
	}

	objectAndFile := bytes.SplitN(split[len(split)-1], []byte("\t"), 2)
	parsedEntry := append(split[:len(split)-1], objectAndFile...)
	if len(parsedEntry) != 4 {
		return nil, ErrParse
	}

	objectType, err := toEnum(string(parsedEntry[1]))
	if err != nil {
		return nil, err
	}

	result := &Entry{}
	result.Mode = parsedEntry[0]
	result.Type = objectType
	result.Oid = string(parsedEntry[2])

	// We know that the last byte in 'path' will be a zero byte.
	result.Path = string(bytes.TrimRight(parsedEntry[3], "\x00"))

	return result, nil
}

func toEnum(s string) (ObjectType, error) {
	switch s {
	case "tree":
		return tree, nil
	case "blob":
		return blob, nil
	case "commit":
		return submodule, nil
	default:
		return -1, ErrParse
	}
}
