package ref

import (
	"bytes"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var localBranchFormatFields = []string{
	"%(refname)", "%(objectname)", "%(contents:subject)", "%(authorname)",
	"%(authoremail)", "%(authordate:iso-strict)", "%(committername)",
	"%(committeremail)", "%(committerdate:iso-strict)",
}

func appendRef(refs [][]byte, p []byte) ([][]byte, int) {
	ref := make([]byte, len(p))
	size := copy(ref, p)
	return append(refs, ref), size
}

type refsSender interface {
	sendRefs([][]byte) error
}

type refsWriter struct {
	refsSender
	MaxMsgSize int
	refsSize   int
	refs       [][]byte
}

func (w *refsWriter) Flush() error {
	if len(w.refs) == 0 { // No message to send, just return
		return nil
	}

	if err := w.refsSender.sendRefs(w.refs); err != nil {
		return err
	}

	// Reset the message
	w.refs = nil
	w.refsSize = 0

	return nil
}

func (w *refsWriter) AddRef(p []byte) error {
	refs, size := appendRef(w.refs, p)
	w.refsSize += size
	w.refs = refs

	if w.refsSize > w.MaxMsgSize {
		return w.Flush()
	}

	return nil
}

type branchesSender struct {
	stream pb.Ref_FindAllBranchNamesServer
}

func (w branchesSender) sendRefs(refs [][]byte) error {
	return w.stream.Send(&pb.FindAllBranchNamesResponse{Names: refs})
}

type tagsSender struct {
	stream pb.Ref_FindAllTagNamesServer
}

func (w tagsSender) sendRefs(refs [][]byte) error {
	return w.stream.Send(&pb.FindAllTagNamesResponse{Names: refs})
}

type localBranchesSender struct {
	stream pb.Ref_FindLocalBranchesServer
}

func buildBranch(elements [][]byte) (*pb.FindLocalBranchResponse, error) {
	authorDate, err := time.Parse(time.RFC3339, string(elements[5]))
	if err != nil {
		return nil, err
	}

	committerDate, err := time.Parse(time.RFC3339, string(elements[8]))
	if err != nil {
		return nil, err
	}

	author := pb.FindLocalBranchCommitAuthor{
		Name:  elements[3],
		Email: elements[4],
		Date:  &timestamp.Timestamp{Seconds: authorDate.Unix()},
	}
	committer := pb.FindLocalBranchCommitAuthor{
		Name:  elements[6],
		Email: elements[7],
		Date:  &timestamp.Timestamp{Seconds: committerDate.Unix()},
	}

	return &pb.FindLocalBranchResponse{
		Name:            elements[0],
		CommitId:        string(elements[1]),
		CommitSubject:   elements[2],
		CommitAuthor:    &author,
		CommitCommitter: &committer,
	}, nil
}

func (w localBranchesSender) sendRefs(refs [][]byte) error {
	var branches []*pb.FindLocalBranchResponse

	for _, ref := range refs {
		elements := bytes.Split(ref, []byte("\x00"))
		if len(elements) != 9 {
			return grpc.Errorf(codes.Internal, "error parsing ref %q", ref)
		}
		branch, err := buildBranch(elements)
		if err != nil {
			return err
		}
		branches = append(branches, branch)
	}
	return w.stream.Send(&pb.FindLocalBranchesResponse{Branches: branches})
}

func newFindAllBranchNamesWriter(stream pb.Ref_FindAllBranchNamesServer, maxMsgSize int) refsWriter {
	return refsWriter{
		refsSender: branchesSender{stream},
		MaxMsgSize: maxMsgSize,
	}
}

func newFindAllTagNamesWriter(stream pb.Ref_FindAllTagNamesServer, maxMsgSize int) refsWriter {
	return refsWriter{
		refsSender: tagsSender{stream},
		MaxMsgSize: maxMsgSize,
	}
}

func newFindLocalBranchesWriter(stream pb.Ref_FindLocalBranchesServer, maxMsgSize int) refsWriter {
	return refsWriter{
		refsSender: localBranchesSender{stream},
		MaxMsgSize: maxMsgSize,
	}
}
