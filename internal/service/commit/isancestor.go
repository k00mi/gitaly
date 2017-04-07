package commit

import (
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"golang.org/x/net/context"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (s *server) CommitIsAncestor(ctx context.Context, in *pb.CommitIsAncestorRequest) (*pb.CommitIsAncestorResponse, error) {
	repoPath, err := helper.GetRepoPath(in.GetRepository())
	if err != nil {
		return nil, err
	}
	if in.AncestorId == "" {
		message := "Bad Request (empty ancestor sha)"
		log.Printf("CommitIsAncestor: %q", message)
		return nil, grpc.Errorf(codes.InvalidArgument, message)
	}
	if in.ChildId == "" {
		message := "Bad Request (empty child sha)"
		log.Printf("CommitIsAncestor: %q", message)
		return nil, grpc.Errorf(codes.InvalidArgument, message)
	}

	ret, err := commitIsAncestorName(repoPath, in.AncestorId, in.ChildId)
	return &pb.CommitIsAncestorResponse{Value: ret}, err
}

// Assumes that `path`, `ancestorID` and `childID` are populated :trollface:
func commitIsAncestorName(path, ancestorID, childID string) (bool, error) {
	cmd := helper.GitCommand("git", "--git-dir", path, "merge-base", "--is-ancestor", ancestorID, childID)

	log.Printf("commitIsAncestor: RepoPath=%q ancestorSha=%s childSha=%s", path, ancestorID, childID)

	if err := cmd.Run(); err != nil {
		if code, ok := helper.ExitStatus(err); ok && code == 1 {
			// This is not really an error, this is `git` saying "This is not an ancestor"
			return false, nil
		}
		return false, grpc.Errorf(codes.Internal, err.Error())
	}

	return true, nil
}
