package git

import (
	"context"
	"errors"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/internal/git/hooks"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitlabshell"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/metadata"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

// WithRefTxHook returns an option that populates the safe command with the
// environment variables necessary to properly execute a reference hook for
// repository changes that may possibly update references
func WithRefTxHook(ctx context.Context, repo *gitalypb.Repository, cfg config.Cfg) CmdOpt {
	return func(cc *cmdCfg) error {
		if repo == nil {
			return fmt.Errorf("missing repo: %w", ErrInvalidArg)
		}

		if err := cc.configureHooks(ctx, repo, cfg); err != nil {
			return fmt.Errorf("ref hook env var: %w", err)
		}

		return nil
	}
}

// configureHooks updates the command configuration to include all environment
// variables required by the reference transaction hook and any other needed
// options to successfully execute hooks.
func (cc *cmdCfg) configureHooks(ctx context.Context, repo *gitalypb.Repository, cfg config.Cfg) error {
	if cc.hooksConfigured {
		return errors.New("hooks already configured")
	}

	payload, err := NewHooksPayload(cfg, repo, nil, nil).Env()
	if err != nil {
		return err
	}

	txEnvs, err := transactionEnv(ctx)
	if err != nil {
		return fmt.Errorf("transaction environment: %w", err)
	}
	cc.env = append(cc.env, txEnvs...)
	cc.env = append(cc.env, payload, "GITALY_BIN_DIR="+cfg.BinDir)

	cc.globals = append(cc.globals, ValueFlag{"-c", fmt.Sprintf("core.hooksPath=%s", hooks.Path(cfg))})
	cc.hooksConfigured = true

	return nil
}

// ReceivePackRequest abstracts away the different requests that end up
// spawning git-receive-pack.
type ReceivePackRequest interface {
	GetGlId() string
	GetGlUsername() string
	GetGlRepository() string
	GetRepository() *gitalypb.Repository
}

// WithReceivePackHooks returns an option that populates the safe command with the environment
// variables necessary to properly execute the pre-receive, update and post-receive hooks for
// git-receive-pack(1).
func WithReceivePackHooks(ctx context.Context, cfg config.Cfg, req ReceivePackRequest, protocol string) CmdOpt {
	return func(cc *cmdCfg) error {
		if err := cc.configureHooks(ctx, req.GetRepository(), config.Config); err != nil {
			return err
		}

		env, err := receivePackHookEnv(ctx, cc, cfg, req, protocol)
		if err != nil {
			return fmt.Errorf("receive-pack hook envvars: %w", err)
		}

		cc.env = append(cc.env, env...)
		return nil
	}
}

func receivePackHookEnv(ctx context.Context, cc *cmdCfg, cfg config.Cfg, req ReceivePackRequest, protocol string) ([]string, error) {
	gitlabshellEnv, err := gitlabshell.EnvFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	env := append(gitlabshellEnv,
		fmt.Sprintf("GL_ID=%s", req.GetGlId()),
		fmt.Sprintf("GL_USERNAME=%s", req.GetGlUsername()),
		fmt.Sprintf("GL_REPOSITORY=%s", req.GetGlRepository()),
		fmt.Sprintf("GL_PROJECT_PATH=%s", req.GetRepository().GetGlProjectPath()),
		fmt.Sprintf("GL_PROTOCOL=%s", protocol),
	)

	return env, nil
}

func transactionEnv(ctx context.Context) ([]string, error) {
	transaction, err := metadata.TransactionFromContext(ctx)
	if err != nil {
		if errors.Is(err, metadata.ErrTransactionNotFound) {
			return nil, nil
		}
		return nil, err
	}

	praefect, err := metadata.PraefectFromContext(ctx)
	if err != nil {
		return nil, err
	}

	praefectEnv, err := praefect.Env()
	if err != nil {
		return nil, err
	}

	transactionEnv, err := transaction.Env()
	if err != nil {
		return nil, err
	}

	return []string{praefectEnv, transactionEnv}, nil
}
