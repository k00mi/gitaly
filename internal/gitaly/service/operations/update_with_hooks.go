package operations

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/gitaly/internal/git/updateref"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/hook"
	"gitlab.com/gitlab-org/gitaly/internal/gitlabshell"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/metadata"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type preReceiveError struct {
	message string
}

func (e preReceiveError) Error() string {
	return e.message
}

type updateRefError struct {
	reference string
}

func (e updateRefError) Error() string {
	return fmt.Sprintf("Could not update %s. Please refresh and try again.", e.reference)
}

func (s *Server) updateReferenceWithHooks(ctx context.Context, repo *gitalypb.Repository, user *gitalypb.User, reference, newrev, oldrev string) error {
	gitlabshellEnv, err := gitlabshell.EnvFromConfig(s.cfg)
	if err != nil {
		return err
	}

	env := append([]string{
		"GL_PROTOCOL=web",
		fmt.Sprintf("GL_ID=%s", user.GetGlId()),
		fmt.Sprintf("GL_USERNAME=%s", user.GetGlUsername()),
		fmt.Sprintf("GL_REPOSITORY=%s", repo.GetGlRepository()),
		fmt.Sprintf("GL_PROJECT_PATH=%s", repo.GetGlProjectPath()),
		fmt.Sprintf("GITALY_SOCKET=" + s.cfg.GitalyInternalSocketPath()),
		fmt.Sprintf("GITALY_REPO=%s", repo),
		fmt.Sprintf("GITALY_TOKEN=%s", s.cfg.Auth.Token),
	}, gitlabshellEnv...)

	transaction, err := metadata.TransactionFromContext(ctx)
	if err != nil {
		if err != metadata.ErrTransactionNotFound {
			return err
		}
	}

	if err == nil {
		praefect, err := metadata.PraefectFromContext(ctx)
		if err != nil {
			return err
		}

		transactionEnv, err := transaction.Env()
		if err != nil {
			return err
		}

		praefectEnv, err := praefect.Env()
		if err != nil {
			return err
		}

		env = append(env, transactionEnv, praefectEnv)
	}

	changes := fmt.Sprintf("%s %s %s\n", oldrev, newrev, reference)
	var stdout, stderr bytes.Buffer

	if err := s.hookManager.PreReceiveHook(ctx, repo, env, strings.NewReader(changes), &stdout, &stderr); err != nil {
		msg := hookErrorFromStdoutAndStderr(stdout.String(), stderr.String())
		return preReceiveError{message: msg}
	}
	if err := s.hookManager.UpdateHook(ctx, repo, reference, oldrev, newrev, env, &stdout, &stderr); err != nil {
		msg := hookErrorFromStdoutAndStderr(stdout.String(), stderr.String())
		return preReceiveError{message: msg}
	}

	// For backwards compatibility with Ruby, we need to only call the reference-transaction
	// hook if the corresponding Ruby feature flag is set.
	if featureflag.IsEnabled(ctx, featureflag.RubyReferenceTransactionHook) {
		if err := s.hookManager.ReferenceTransactionHook(ctx, hook.ReferenceTransactionPrepared, env, strings.NewReader(changes)); err != nil {
			return preReceiveError{message: err.Error()}
		}
	}

	updater, err := updateref.New(ctx, repo)
	if err != nil {
		return err
	}

	if err := updater.Update(reference, newrev, oldrev); err != nil {
		return err
	}

	if err := updater.Wait(); err != nil {
		return updateRefError{reference: reference}
	}

	if err := s.hookManager.PostReceiveHook(ctx, repo, nil, env, strings.NewReader(changes), &stdout, &stderr); err != nil {
		msg := hookErrorFromStdoutAndStderr(stdout.String(), stderr.String())
		return preReceiveError{message: msg}
	}

	return nil
}
