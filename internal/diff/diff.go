package diff

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

// Diff represents a single parsed diff entry
type Diff struct {
	FromID    string
	ToID      string
	OldMode   int32
	NewMode   int32
	FromPath  []byte
	ToPath    []byte
	Binary    bool
	RawChunks [][]byte
}

// Parser holds necessary state for parsing a diff stream
type Parser struct {
	reader      *bufio.Reader
	currentDiff *Diff
	finished    bool
	err         error
}

var (
	diffHeaderRegexp       = regexp.MustCompile(`(?m)^diff --git "?a/(.*?)"? "?b/(.*?)"?$`)
	indexHeaderRegexp      = regexp.MustCompile(`(?m)^index ([[:xdigit:]]{40})..([[:xdigit:]]{40})(?:\s([[:digit:]]+))?$`)
	renameCopyHeaderRegexp = regexp.MustCompile(`(?m)^(copy|rename) (from|to) "?(.*?)"?$`)
	modeHeaderRegexp       = regexp.MustCompile(`(?m)^(old|new|(?:deleted|new) file) mode (\d+)$`)
)

// NewDiffParser returns a new Parser
func NewDiffParser(src io.Reader) *Parser {
	reader := bufio.NewReader(src)
	return &Parser{reader: reader}
}

// Parse parses a single diff. It returns true if successful, false if it finished
// parsing all diffs or when it encounters an error, in which case use Parser.Err()
// to get the error.
func (parser *Parser) Parse() bool {
	if parser.finished {
		return false
	}

	parsingDiff := false

	for {
		// We cannot use bufio.Scanner because the line may be very long.
		line, err := parser.reader.Peek(10)
		if err == io.EOF {
			parser.finished = true

			if parser.currentDiff == nil { // Probably NewDiffParser was passed an empty src (e.g. git diff failed)
				return false
			}

			if len(line) > 0 && len(line) < 10 {
				consumeChunkLine(parser.reader, parser.currentDiff)
			}

			return true
		} else if err != nil {
			parser.err = fmt.Errorf("peek diff line: %v", err)
			return false
		}

		if bytes.HasPrefix(line, []byte("diff --git")) {
			if parsingDiff {
				return true
			}

			parser.currentDiff = &Diff{}
			parsingDiff = true

			parser.err = parseHeader(parser.reader, parser.currentDiff)
		} else if bytes.HasPrefix(line, []byte("Binary")) {
			parser.err = consumeBinaryNotice(parser.reader, parser.currentDiff)
		} else if bytes.HasPrefix(line, []byte("@@")) {
			parser.currentDiff.RawChunks = append(parser.currentDiff.RawChunks, nil)

			parser.err = consumeChunkLine(parser.reader, parser.currentDiff)
		} else if helper.ByteSliceHasAnyPrefix(line, "---", "+++") {
			parser.err = consumeLine(parser.reader)
		} else if helper.ByteSliceHasAnyPrefix(line, "-", "+", " ", "\\") {
			parser.err = consumeChunkLine(parser.reader, parser.currentDiff)
		} else {
			parser.err = parseHeader(parser.reader, parser.currentDiff)
		}

		if parser.err != nil {
			return false
		}
	}

	return true
}

// Diff returns a successfully parsed diff. It should be called only when Parser.Parse()
// returns true.
func (parser *Parser) Diff() *Diff {
	return parser.currentDiff
}

// Err returns the error encountered (if any) when parsing the diff stream. It should be called only when Parser.Parse()
// returns false.
func (parser *Parser) Err() error {
	return parser.err
}

func parseHeader(reader *bufio.Reader, diff *Diff) error {
	line, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("read diff header line: %v", err)
	}

	if matches := diffHeaderRegexp.FindSubmatch(line); len(matches) > 0 { // diff --git a/Makefile b/Makefile
		diff.FromPath = unescapeOctalBytes(matches[1])
		diff.ToPath = unescapeOctalBytes(matches[1])
	}

	if matches := indexHeaderRegexp.FindStringSubmatch(string(line)); len(matches) > 0 { // index a8b75d25da09b92b9f8b02151b001217ec24e0ea..3ecb2f9d50ed85f781569431df9f110bff6cb607 100644
		diff.FromID = matches[1]
		diff.ToID = matches[2]
		if matches[3] != "" { // mode does not exist for deleted/new files on this line
			mode, err := strconv.ParseInt(matches[3], 8, 0)
			if err != nil {
				return fmt.Errorf("index header: %v", err)
			}

			diff.OldMode = int32(mode)
			diff.NewMode = int32(mode)
		}
	}

	if matches := renameCopyHeaderRegexp.FindSubmatch(line); len(matches) > 0 { // rename from cmd/gitaly-client/main.go
		switch string(matches[2]) {
		case "from":
			diff.FromPath = unescapeOctalBytes(matches[3])
		case "to":
			diff.ToPath = unescapeOctalBytes(matches[3])
		}
	}

	if matches := modeHeaderRegexp.FindStringSubmatch(string(line)); len(matches) > 0 { // deleted file mode 100644
		mode, err := strconv.ParseInt(matches[2], 8, 0)
		if err != nil {
			return fmt.Errorf("mode header: %v", err)
		}

		switch matches[1] {
		case "old", "deleted file":
			diff.OldMode = int32(mode)
		case "new", "new file":
			diff.NewMode = int32(mode)
		}
	}

	return nil
}

func consumeChunkLine(reader *bufio.Reader, diff *Diff) error {
	line, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("read chunk line: %v", err)
	}

	chunkIndex := len(diff.RawChunks) - 1
	diff.RawChunks[chunkIndex] = append(diff.RawChunks[chunkIndex], line...)

	return nil
}

func consumeBinaryNotice(reader *bufio.Reader, diff *Diff) error {
	_, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("read binary notice: %v", err)
	}

	diff.Binary = true

	return nil
}

func consumeLine(reader *bufio.Reader) error {
	_, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("read line: %v", err)
	}

	return nil
}

func unescapeOctalBytes(s []byte) []byte {
	var unescaped []byte

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) && helper.IsNumber(s[i+1:i+4]) {
			octalByte, err := strconv.ParseUint(string(s[i+1:i+4]), 8, 8)
			if err == nil {
				unescaped = append(unescaped, byte(octalByte))

				i += 3
				continue
			}
		}

		unescaped = append(unescaped, s[i])
	}

	return unescaped
}
