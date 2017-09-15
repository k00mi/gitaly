package git

import (
	"context"
	"os/exec"

	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/helper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

// Command creates a git.Command with the given args
func Command(ctx context.Context, repo *pb.Repository, args ...string) (*command.Command, error) {
	repoPath, err := helper.GetRepoPath(repo)
	if err != nil {
		return nil, err
	}
	args = append([]string{"--git-dir", repoPath}, args...)

	return command.New(ctx, exec.Command(command.GitPath(), args...), nil, nil, nil)
}
