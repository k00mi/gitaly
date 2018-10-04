package git

import (
	"context"
	"os/exec"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git/alternates"
)

// Command creates a git.Command with the given args
func Command(ctx context.Context, repo *gitalypb.Repository, args ...string) (*command.Command, error) {
	repoPath, env, err := alternates.PathAndEnv(repo)
	if err != nil {
		return nil, err
	}

	args = append([]string{"--git-dir", repoPath}, args...)
	return command.New(ctx, exec.Command(command.GitPath(), args...), nil, nil, nil, env...)
}

// CommandWithoutRepo works like Command but without a git repository
func CommandWithoutRepo(ctx context.Context, args ...string) (*command.Command, error) {
	return command.New(ctx, exec.Command(command.GitPath(), args...), nil, nil, nil, "")
}
