package ref

import (
	"bytes"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/git/catfile"
	"gitlab.com/gitlab-org/gitaly/internal/git/log"
	"gitlab.com/gitlab-org/gitaly/internal/helper/lines"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var localBranchFormatFields = []string{"%(refname)", "%(objectname)"}

func parseRef(ref []byte) ([][]byte, error) {
	elements := bytes.Split(ref, []byte("\x00"))
	if len(elements) != len(localBranchFormatFields) {
		return nil, status.Errorf(codes.Internal, "error parsing ref %q", ref)
	}
	return elements, nil
}

func buildLocalBranch(c *catfile.Batch, elements [][]byte) (*pb.FindLocalBranchResponse, error) {
	target, err := log.GetCommitCatfile(c, string(elements[1]))
	if err != nil {
		return nil, err
	}
	author := pb.FindLocalBranchCommitAuthor{
		Name:  target.Author.Name,
		Email: target.Author.Email,
		Date:  target.Author.Date,
	}
	committer := pb.FindLocalBranchCommitAuthor{
		Name:  target.Committer.Name,
		Email: target.Committer.Email,
		Date:  target.Committer.Date,
	}

	return &pb.FindLocalBranchResponse{
		Name:            elements[0],
		CommitId:        target.Id,
		CommitSubject:   target.Subject,
		CommitAuthor:    &author,
		CommitCommitter: &committer,
	}, nil
}

func buildBranch(c *catfile.Batch, elements [][]byte) (*pb.FindAllBranchesResponse_Branch, error) {
	target, err := log.GetCommitCatfile(c, string(elements[1]))
	if err != nil {
		return nil, err
	}

	return &pb.FindAllBranchesResponse_Branch{
		Name:   elements[0],
		Target: target,
	}, nil
}

func newFindAllBranchNamesWriter(stream pb.Ref_FindAllBranchNamesServer) lines.Sender {
	return func(refs [][]byte) error {
		return stream.Send(&pb.FindAllBranchNamesResponse{Names: refs})
	}
}

func newFindAllTagNamesWriter(stream pb.Ref_FindAllTagNamesServer) lines.Sender {
	return func(refs [][]byte) error {
		return stream.Send(&pb.FindAllTagNamesResponse{Names: refs})
	}
}

func newFindLocalBranchesWriter(stream pb.Ref_FindLocalBranchesServer, c *catfile.Batch) lines.Sender {
	return func(refs [][]byte) error {
		var branches []*pb.FindLocalBranchResponse

		for _, ref := range refs {
			elements, err := parseRef(ref)
			if err != nil {
				return err
			}
			branch, err := buildLocalBranch(c, elements)
			if err != nil {
				return err
			}
			branches = append(branches, branch)
		}
		return stream.Send(&pb.FindLocalBranchesResponse{Branches: branches})
	}
}

func newFindAllBranchesWriter(stream pb.RefService_FindAllBranchesServer, c *catfile.Batch) lines.Sender {
	return func(refs [][]byte) error {
		var branches []*pb.FindAllBranchesResponse_Branch

		for _, ref := range refs {
			elements, err := parseRef(ref)
			if err != nil {
				return err
			}
			branch, err := buildBranch(c, elements)
			if err != nil {
				return err
			}
			branches = append(branches, branch)
		}
		return stream.Send(&pb.FindAllBranchesResponse{Branches: branches})
	}
}
