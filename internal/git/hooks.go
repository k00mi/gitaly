package git

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

// WithRefTxHook returns an option that populates the safe command with the
// environment variables necessary to properly execute a reference hook for
// repository changes that may possibly update references
func WithRefTxHook(ctx context.Context, repo *gitalypb.Repository, cfg config.Cfg) CmdOpt {
	return func(cc *cmdCfg) error {
		rfEnvs, err := refHookEnv(ctx, repo, cfg)
		if err != nil {
			return fmt.Errorf("ref hook env var: %w", err)
		}

		cc.env = append(cc.env, rfEnvs...)
		return nil
	}
}

// refHookEnv returns all env vars required by the reference transaction hook
func refHookEnv(ctx context.Context, repo *gitalypb.Repository, cfg config.Cfg) ([]string, error) {
	repoJSON, err := jsonpbMarshaller.MarshalToString(repo)
	if err != nil {
		return nil, err
	}

	return []string{
		"GITALY_SOCKET=" + config.GitalyInternalSocketPath(),
		fmt.Sprintf("GITALY_REPO=%s", repoJSON),
		fmt.Sprintf("GITALY_TOKEN=%s", cfg.Auth.Token),
		fmt.Sprintf("%s=%v", featureflag.ReferenceTransactionHookEnvVar, featureflag.IsEnabled(ctx, featureflag.ReferenceTransactionHook)),
	}, nil
}
