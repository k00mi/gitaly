package git

import (
	"strings"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/command"
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

	opt := WithRefTxHook(ctx, testRepo, config.Config)
	subCmd := SubCmd{Name: "update-ref", Args: []string{"refs/heads/master", NullSHA}}

	for _, tt := range []struct {
		name string
		fn   func() (*command.Command, error)
	}{
		{
			name: "SafeCmd",
			fn: func() (*command.Command, error) {
				return SafeCmd(ctx, testRepo, nil, subCmd, opt)
			},
		},
		{
			name: "SafeCmdWithEnv",
			fn: func() (*command.Command, error) {
				return SafeCmdWithEnv(ctx, nil, testRepo, nil, subCmd, opt)
			},
		},
		{
			name: "SafeStdinCmd",
			fn: func() (*command.Command, error) {
				return SafeStdinCmd(ctx, testRepo, nil, subCmd, opt)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := tt.fn()
			require.NoError(t, err)
			// There is no full setup, so executing the hook will fail.
			require.Error(t, cmd.Wait())

			var actualEnvVars []string
			for _, env := range cmd.Env() {
				kv := strings.SplitN(env, "=", 2)
				require.Len(t, kv, 2)
				key, val := kv[0], kv[1]

				if strings.HasPrefix(key, "GL_") || strings.HasPrefix(key, "GITALY_") {
					require.NotEmptyf(t, strings.TrimSpace(val),
						"env var %s value should not be empty string", key)
					actualEnvVars = append(actualEnvVars, key)
				}
			}

			require.EqualValues(t, []string{
				"GITALY_BIN_DIR",
				"GITALY_SOCKET",
				"GITALY_REPO",
				"GITALY_TOKEN",
				"GITALY_REFERENCE_TRANSACTION_HOOK",
			}, actualEnvVars)
		})
	}
}

func TestHooksPayload(t *testing.T) {
	repo, _, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	marshaller := &jsonpb.Marshaler{}
	marshalledRepo, err := marshaller.MarshalToString(repo)
	require.NoError(t, err)

	t.Run("envvar has proper name", func(t *testing.T) {
		env, err := NewHooksPayload(config.Config, repo).Env()
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(env, ErrHooksPayloadNotFound+"="))
	})

	t.Run("roundtrip succeeds", func(t *testing.T) {
		env, err := NewHooksPayload(config.Config, repo).Env()
		require.NoError(t, err)

		payload, err := HooksPayloadFromEnv([]string{
			"UNRELATED=value",
			env,
			"ANOTHOR=unrelated-value",
			ErrHooksPayloadNotFound + "_WITH_SUFFIX=is-ignored",
		})
		require.NoError(t, err)

		require.Equal(t, HooksPayload{
			Repo:           repo,
			BinDir:         config.Config.BinDir,
			InternalSocket: config.Config.GitalyInternalSocketPath(),
		}, payload)
	})

	t.Run("missing envvar", func(t *testing.T) {
		_, err := HooksPayloadFromEnv([]string{"OTHER_ENV=foobar"})
		require.Error(t, err)
		require.Equal(t, ErrPayloadNotFound, err)
	})

	t.Run("bogus value", func(t *testing.T) {
		_, err := HooksPayloadFromEnv([]string{ErrHooksPayloadNotFound + "=foobar"})
		require.Error(t, err)
	})

	t.Run("fallback", func(t *testing.T) {
		payload, err := HooksPayloadFromEnv([]string{
			"GITALY_BIN_DIR=/bin/dir",
			"GITALY_SOCKET=/path/to/socket",
			"GITALY_TOKEN=secret",
			"GITALY_REPOSITORY=" + marshalledRepo,
		})
		require.NoError(t, err)

		require.Equal(t, HooksPayload{
			Repo:                repo,
			BinDir:              "/bin/dir",
			InternalSocket:      "/path/to/socket",
			InternalSocketToken: "secret",
		}, payload)
	})

	t.Run("fallback without token", func(t *testing.T) {
		payload, err := HooksPayloadFromEnv([]string{
			"GITALY_BIN_DIR=/bin/dir",
			"GITALY_SOCKET=/path/to/socket",
			"GITALY_REPOSITORY=" + marshalledRepo,
		})
		require.NoError(t, err)

		require.Equal(t, HooksPayload{
			Repo:           repo,
			BinDir:         "/bin/dir",
			InternalSocket: "/path/to/socket",
		}, payload)
	})

	t.Run("fallback with missing repository", func(t *testing.T) {
		_, err := HooksPayloadFromEnv([]string{
			"GITALY_BIN_DIR=/bin/dir",
			"GITALY_SOCKET=/path/to/socket",
		})
		require.Equal(t, ErrPayloadNotFound, err)
	})

	t.Run("fallback with missing bindir", func(t *testing.T) {
		_, err := HooksPayloadFromEnv([]string{
			"GITALY_SOCKET=/path/to/socket",
			"GITALY_REPOSITORY=" + marshalledRepo,
		})
		require.Equal(t, ErrPayloadNotFound, err)
	})

	t.Run("fallback with missing socket", func(t *testing.T) {
		_, err := HooksPayloadFromEnv([]string{
			"GITALY_BIN_DIR=/bin/dir",
			"GITALY_REPOSITORY=" + marshalledRepo,
		})
		require.Equal(t, ErrPayloadNotFound, err)
	})
}
