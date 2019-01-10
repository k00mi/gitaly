package rawdiff

import (
	"bufio"
	"fmt"
	"io"
)

// Diff represents a `git diff --raw` entry.
type Diff struct {
	// The naming of the fields below follows the "RAW DIFF FORMAT" of `git
	// diff`.

	SrcMode string
	DstMode string
	SrcSHA  string
	DstSHA  string
	Status  string
	SrcPath string
	DstPath string // optional!
}

// Parser is a parser for the "-z" variant of the "RAW DIFF FORMAT"
// documented in `git help diff`.
type Parser struct {
	r *bufio.Reader
}

// NewParser returns a new Parser instance. The reader must contain
// output from `git diff --raw -z`.
func NewParser(r io.Reader) *Parser {
	return &Parser{r: bufio.NewReader(r)}
}

// NextDiff returns the next raw diff. If there are no more diffs, the
// error is io.EOF.
func (p *Parser) NextDiff() (*Diff, error) {
	c, err := p.r.ReadByte()
	if err != nil {
		return nil, err
	}

	if c != ':' {
		return nil, fmt.Errorf("expected leading colon in raw diff line")
	}

	d := &Diff{}

	for _, field := range []*string{&d.SrcMode, &d.DstMode, &d.SrcSHA, &d.DstSHA} {
		if *field, err = p.readStringChop(' '); err != nil {
			return nil, err
		}
	}

	for _, field := range []*string{&d.Status, &d.SrcPath} {
		if *field, err = p.readStringChop(0); err != nil {
			return nil, err
		}
	}

	if len(d.Status) > 0 && (d.Status[0] == 'C' || d.Status[0] == 'R') {
		if d.DstPath, err = p.readStringChop(0); err != nil {
			return nil, err
		}
	}

	return d, nil
}

// readStringChop combines bufio.Reader.ReadString with removing the
// trailing delimiter.
func (p *Parser) readStringChop(delim byte) (string, error) {
	s, err := p.r.ReadString(delim)
	if err != nil {
		return "", fmt.Errorf("read raw diff: %v", err)
	}

	return s[:len(s)-1], nil
}
