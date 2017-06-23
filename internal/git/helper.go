package git

import (
	"bufio"
	"io"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

var (
	// CommitFormatFields specifies the fields used to parse commits
	CommitFormatFields = []string{
		"%(objectname)", "%(contents:subject)", "%(authorname)",
		"%(authoremail)", "%(authordate:iso-strict)", "%(committername)",
		"%(committeremail)", "%(committerdate:iso-strict)",
	}

	// CommitLogFormatFields translates `CommitFormatFields` to `git log` format
	CommitLogFormatFields = []string{
		"%H", "%s", "%an", "%ae", "%aI", "%cn", "%ce", "%cI",
	}
)

// RefsSender implementers take care of handling `RefsWriter`s refs buffers
type RefsSender interface {
	SendRefs([][]byte) error
}

// RefsWriter abstracts a sender of refs
type RefsWriter struct {
	RefsSender
	MaxMsgSize int
	refsSize   int
	refs       [][]byte
}

// AppendRef adds a ref to the `refs` array, making sure the element added is a
// copy of the input, to avoid io buffer shennanigans
func AppendRef(refs [][]byte, p []byte) ([][]byte, int) {
	ref := make([]byte, len(p))
	size := copy(ref, p)
	return append(refs, ref), size
}

// Flush calls the RefsSender `SendRefs` method with the accumulated refs and
// clears the refs buffer.
func (w *RefsWriter) Flush() error {
	if len(w.refs) == 0 { // No message to send, just return
		return nil
	}

	if err := w.RefsSender.SendRefs(w.refs); err != nil {
		return err
	}

	// Reset the message
	w.refs = nil
	w.refsSize = 0

	return nil
}

// AddRef adds a new ref to the RefsWriter buffer, and flushes if the maximum
// size has been achieved
func (w *RefsWriter) AddRef(p []byte) error {
	refs, size := AppendRef(w.refs, p)
	w.refsSize += size
	w.refs = refs

	if w.refsSize > w.MaxMsgSize {
		return w.Flush()
	}

	return nil
}

// HandleGitCommand reads from an `io.Reader` coming from a git command and
// writes each line as a ref to the given `RefsWriter`
func HandleGitCommand(r io.Reader, w RefsWriter) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := w.AddRef(scanner.Bytes()); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return w.Flush()
}

// BuildCommit builds a commit based on an array of commit elements
func BuildCommit(elements [][]byte) (*pb.GitCommit, error) {
	authorDate, err := time.Parse(time.RFC3339, string(elements[4]))
	if err != nil {
		return nil, err
	}

	committerDate, err := time.Parse(time.RFC3339, string(elements[7]))
	if err != nil {
		return nil, err
	}

	author := pb.CommitAuthor{
		Name:  elements[2],
		Email: elements[3],
		Date:  &timestamp.Timestamp{Seconds: authorDate.Unix()},
	}
	committer := pb.CommitAuthor{
		Name:  elements[5],
		Email: elements[6],
		Date:  &timestamp.Timestamp{Seconds: committerDate.Unix()},
	}

	return &pb.GitCommit{
		Id:        string(elements[0]),
		Subject:   elements[1],
		Author:    &author,
		Committer: &committer,
	}, nil
}
