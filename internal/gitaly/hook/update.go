package hook

import (
	"context"
	"io"

	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func (m *GitLabHookManager) UpdateHook(ctx context.Context, repo *gitalypb.Repository, ref, oldValue, newValue string, env []string, stdout, stderr io.Writer) error {
	payload, err := git.HooksPayloadFromEnv(env)
	if err != nil {
		return helper.ErrInternalf("extracting hooks payload: %w", err)
	}

	if !isPrimary(payload) {
		return nil
	}

	if ref == "" {
		return helper.ErrInternalf("hook got no reference")
	}
	if err := git.ValidateCommitID(oldValue); err != nil {
		return helper.ErrInternalf("hook got invalid old value: %w", err)
	}
	if err := git.ValidateCommitID(newValue); err != nil {
		return helper.ErrInternalf("hook got invalid new value: %w", err)
	}
	if payload.ReceiveHooksPayload == nil {
		return helper.ErrInternalf("payload has no receive hooks info")
	}

	executor, err := m.newCustomHooksExecutor(repo, "update")
	if err != nil {
		return helper.ErrInternal(err)
	}

	if err = executor(
		ctx,
		[]string{ref, oldValue, newValue},
		append(env, customHooksEnv(payload, nil)...),
		nil,
		stdout,
		stderr,
	); err != nil {
		return err
	}

	return nil
}
