package repository

import (
	"io/ioutil"
	"strings"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/git"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) FindMergeBase(ctx context.Context, req *pb.FindMergeBaseRequest) (*pb.FindMergeBaseResponse, error) {
	revisions := req.GetRevisions()
	if len(revisions) < 2 {
		return nil, status.Errorf(codes.InvalidArgument, "FindMergeBase: at least 2 revisions are required")
	}

	args := []string{"merge-base"}
	for _, revision := range revisions {
		args = append(args, string(revision))
	}

	cmd, err := git.Command(ctx, req.GetRepository(), args...)
	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "FindMergeBase: cmd: %v", err)
	}

	mergeBase, err := ioutil.ReadAll(cmd)
	if err != nil {
		return nil, err
	}

	mergeBaseStr := strings.TrimSpace(string(mergeBase))

	if err := cmd.Wait(); err != nil {
		// On error just return an empty merge base
		return &pb.FindMergeBaseResponse{Base: ""}, nil
	}

	return &pb.FindMergeBaseResponse{Base: mergeBaseStr}, nil
}
