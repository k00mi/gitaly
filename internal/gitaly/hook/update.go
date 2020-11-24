package hook

import (
	"context"
	"io"

	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func (m *GitLabHookManager) UpdateHook(ctx context.Context, repo *gitalypb.Repository, ref, oldValue, newValue string, env []string, stdout, stderr io.Writer) error {
	primary, err := isPrimary(env)
	if err != nil {
		return helper.ErrInternalf("could not check role: %w", err)
	}
	if !primary {
		return nil
	}

	executor, err := m.newCustomHooksExecutor(repo, "update")
	if err != nil {
		return helper.ErrInternal(err)
	}

	if err = executor(
		ctx,
		[]string{ref, oldValue, newValue},
		env,
		nil,
		stdout,
		stderr,
	); err != nil {
		return err
	}

	return nil
}
