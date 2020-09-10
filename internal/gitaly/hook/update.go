package hook

import (
	"context"
	"io"

	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func (m *GitLabHookManager) UpdateHook(ctx context.Context, repo *gitalypb.Repository, ref, oldValue, newValue string, env []string, stdout, stderr io.Writer) error {
	repoPath, err := helper.GetRepoPath(repo)
	if err != nil {
		return err
	}

	executor, err := m.NewCustomHooksExecutor(repoPath, config.Config.Hooks.CustomHooksDir, "update")
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
