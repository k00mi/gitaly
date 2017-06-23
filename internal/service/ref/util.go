package ref

import (
	"bytes"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var localBranchFormatFields = append([]string{"%(refname)"}, git.CommitFormatFields...)

type branchesSender struct {
	stream pb.Ref_FindAllBranchNamesServer
}

func (w branchesSender) SendRefs(refs [][]byte) error {
	return w.stream.Send(&pb.FindAllBranchNamesResponse{Names: refs})
}

type tagsSender struct {
	stream pb.Ref_FindAllTagNamesServer
}

func (w tagsSender) SendRefs(refs [][]byte) error {
	return w.stream.Send(&pb.FindAllTagNamesResponse{Names: refs})
}

type localBranchesSender struct {
	stream pb.Ref_FindLocalBranchesServer
}

func buildBranch(elements [][]byte) (*pb.FindLocalBranchResponse, error) {
	target, err := git.BuildCommit(elements[1:])

	if err != nil {
		return nil, err
	}

	return &pb.FindLocalBranchResponse{
		Name:   elements[0],
		Target: target,
	}, nil
}

func (w localBranchesSender) SendRefs(refs [][]byte) error {
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

func newFindAllBranchNamesWriter(stream pb.Ref_FindAllBranchNamesServer, maxMsgSize int) git.RefsWriter {
	return git.RefsWriter{
		RefsSender: branchesSender{stream},
		MaxMsgSize: maxMsgSize,
	}
}

func newFindAllTagNamesWriter(stream pb.Ref_FindAllTagNamesServer, maxMsgSize int) git.RefsWriter {
	return git.RefsWriter{
		RefsSender: tagsSender{stream},
		MaxMsgSize: maxMsgSize,
	}
}

func newFindLocalBranchesWriter(stream pb.Ref_FindLocalBranchesServer, maxMsgSize int) git.RefsWriter {
	return git.RefsWriter{
		RefsSender: localBranchesSender{stream},
		MaxMsgSize: maxMsgSize,
	}
}
