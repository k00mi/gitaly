package git_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestWithRefHook(t *testing.T) {
	testRepo, _, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	ctx, cancel := testhelper.Context()
	defer cancel()

	const token = "my-super-secure-token"
	defer func(oldToken string) { config.Config.Auth.Token = oldToken }(config.Config.Auth.Token)
	config.Config.Auth.Token = token

	opt := git.WithRefTxHook(ctx, testRepo, config.Config)
	subCmd := git.SubCmd{Name: "update-ref", Args: []string{"refs/heads/master", git.NullSHA}}

	for _, tt := range []struct {
		name string
		fn   func() (*command.Command, error)
	}{
		{
			name: "SafeCmd",
			fn: func() (*command.Command, error) {
				return git.SafeCmd(ctx, testRepo, nil, subCmd, opt)
			},
		},
		{
			name: "SafeCmdWithEnv",
			fn: func() (*command.Command, error) {
				return git.SafeCmdWithEnv(ctx, nil, testRepo, nil, subCmd, opt)
			},
		},
		{
			name: "SafeStdinCmd",
			fn: func() (*command.Command, error) {
				return git.SafeStdinCmd(ctx, testRepo, nil, subCmd, opt)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			expectEnvVars := map[string]struct{}{
				"GITALY_BIN_DIR":                    struct{}{},
				"GITALY_SOCKET":                     struct{}{},
				"GITALY_REPO":                       struct{}{},
				"GITALY_TOKEN":                      struct{}{},
				"GITALY_REFERENCE_TRANSACTION_HOOK": struct{}{},
			}

			cmd, err := tt.fn()
			require.NoError(t, err)
			// There is no full setup, so executing the hook will fail.
			require.Error(t, cmd.Wait())

			var actualEnvVars []string
			for _, env := range cmd.Env() {
				kv := strings.SplitN(env, "=", 2)
				if len(kv) < 2 {
					continue
				}
				key, val := kv[0], kv[1]

				if _, ok := expectEnvVars[key]; ok {
					assert.NotEmptyf(t, strings.TrimSpace(val),
						"env var %s value should not be empty string", key)
				}
				actualEnvVars = append(actualEnvVars, key)
			}

			for k := range expectEnvVars {
				assert.Contains(t, actualEnvVars, k)
			}
		})
	}
}
