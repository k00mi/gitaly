package commit

import (
	"fmt"

	"gitlab.com/gitlab-org/gitaly/internal/git/ls_tree"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) ListLastCommitsForTree(ctx context.Context, in *pb.ListLastCommitsForTreeRequest) (*pb.ListLastCommitsForTreeResponse, error) {
	if err := validateListLastCommitsForTreeRequest(in); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "ListLastCommitsForTree: %v", err)
	}

	path := string(in.GetPath())
	if len(path) == 0 || path == "/" {
		path = "."
	}

	commits, err := ls_tree.LastCommitsForTree(ctx, in.GetRepository(), string(in.GetRevision()), path)
	if err != nil {
		return nil, err
	}

	return &pb.ListLastCommitsForTreeResponse{Commits: commits}, nil
}

func validateListLastCommitsForTreeRequest(in *pb.ListLastCommitsForTreeRequest) error {
	if len(in.Revision) == 0 {
		return fmt.Errorf("empty Revision")
	}
	return nil
}
