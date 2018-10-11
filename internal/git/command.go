package git

import (
	"context"
	"io"
	"os/exec"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git/alternates"
)

// Command creates a git.Command with the given args and Repository
func Command(ctx context.Context, repo *gitalypb.Repository, args ...string) (*command.Command, error) {
	repoPath, env, err := alternates.PathAndEnv(repo)

	if err != nil {
		return nil, err
	}

	args = append([]string{"--git-dir", repoPath}, args...)

	return BareCommand(ctx, nil, nil, nil, env, args...)
}

// BareCommand creates a git.Command with the given args, stdin/stdout/stderr, and env
func BareCommand(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, env []string, args ...string) (*command.Command, error) {
	env = append(env, command.GitEnv...)

	return command.New(ctx, exec.Command(command.GitPath(), args...), stdin, stdout, stderr, env...)
}

// CommandWithoutRepo works like Command but without a git repository
func CommandWithoutRepo(ctx context.Context, args ...string) (*command.Command, error) {
	return BareCommand(ctx, nil, nil, nil, nil, args...)
}
