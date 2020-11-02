package git

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/jsonpb"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

var jsonpbMarshaller = &jsonpb.Marshaler{}

// WithRefTxHook returns an option that populates the safe command with the
// environment variables necessary to properly execute a reference hook for
// repository changes that may possibly update references
func WithRefTxHook(ctx context.Context, repo *gitalypb.Repository, cfg config.Cfg) CmdOpt {
	return func(cc *cmdCfg) error {
		if repo == nil {
			return fmt.Errorf("missing repo: %w", ErrInvalidArg)
		}

		rfEnvs, err := refHookEnv(ctx, repo, cfg)
		if err != nil {
			return fmt.Errorf("ref hook env var: %w", err)
		}

		cc.env = append(cc.env, rfEnvs...)
		cc.refHookConfigured = true

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
		fmt.Sprintf("%s=true", featureflag.ReferenceTransactionHookEnvVar),
	}, nil
}

type refHookRequired struct{}

// RequireRefHook updates the context to indicate a ref hook is required for
// the current operation
func RequireRefHook(ctx context.Context) context.Context {
	return context.WithValue(ctx, refHookRequired{}, true)
}

// IsRefHookRequired returns true if the context has been marked to indicate a
// ref hook may be required
func IsRefHookRequired(ctx context.Context) bool {
	return ctx.Value(refHookRequired{}) != nil
}
