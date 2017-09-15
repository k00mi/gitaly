package ref

import (
	"context"

	"gitlab.com/gitlab-org/gitaly/internal/git"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

// IsValidRef checks if a ref in a repo is valid
func IsValidRef(ctx context.Context, repo *pb.Repository, ref string) bool {
	if ref == "" {
		return false
	}

	cmd, err := git.Command(ctx, repo, "log", "--max-count=1", ref)
	if err != nil {
		return false
	}

	return cmd.Wait() == nil
}
