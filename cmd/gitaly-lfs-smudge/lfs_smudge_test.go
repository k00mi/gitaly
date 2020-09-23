package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

const (
	lfsOid     = "3ea5dd307f195f449f0e08234183b82e92c3d5f4cff11c2a6bb014f9e0de12aa"
	lfsPointer = `version https://git-lfs.github.com/spec/v1
oid sha256:3ea5dd307f195f449f0e08234183b82e92c3d5f4cff11c2a6bb014f9e0de12aa
size 177735
`
	glRepository = "project-1"
	secretToken  = "topsecret"
	testData     = "hello world"
)

var (
	defaultOptions = testhelper.GitlabTestServerOptions{
		SecretToken:  secretToken,
		LfsBody:      testData,
		LfsOid:       lfsOid,
		GlRepository: glRepository,
	}
)

type mapConfig struct {
	env map[string]string
}

func (m *mapConfig) Get(key string) string {
	return m.env[key]
}

func runTestServer(t *testing.T, options testhelper.GitlabTestServerOptions) (config.Gitlab, func()) {
	tempDir, cleanup := testhelper.TempDir(t)

	testhelper.WriteShellSecretFile(t, tempDir, secretToken)
	secretFilePath := filepath.Join(tempDir, ".gitlab_shell_secret")

	serverURL, serverCleanup := testhelper.NewGitlabTestServer(t, options)

	c := config.Gitlab{URL: serverURL, SecretFile: secretFilePath}

	return c, func() {
		cleanup()
		serverCleanup()
	}
}

func TestSuccessfulLfsSmudge(t *testing.T) {
	var b bytes.Buffer
	reader := strings.NewReader(lfsPointer)

	c, cleanup := runTestServer(t, defaultOptions)
	defer cleanup()

	cfg, err := json.Marshal(c)
	require.NoError(t, err)

	env := map[string]string{
		"GL_REPOSITORY":      "project-1",
		"GL_INTERNAL_CONFIG": string(cfg),
	}
	cfgProvider := &mapConfig{env: env}

	err = smudge(&b, reader, cfgProvider)
	require.NoError(t, err)
	require.Equal(t, testData, b.String())
}

func TestUnsuccessfulLfsSmudge(t *testing.T) {
	testCases := []struct {
		desc          string
		data          string
		missingEnv    string
		expectedError bool
		options       testhelper.GitlabTestServerOptions
	}{
		{
			desc:          "bad LFS pointer",
			data:          "test data",
			options:       defaultOptions,
			expectedError: false,
		},
		{
			desc:          "missing GL_REPOSITORY",
			data:          lfsPointer,
			missingEnv:    "GL_REPOSITORY",
			options:       defaultOptions,
			expectedError: true,
		},
		{
			desc:          "missing GL_INTERNAL_CONFIG",
			data:          lfsPointer,
			missingEnv:    "GL_INTERNAL_CONFIG",
			options:       defaultOptions,
			expectedError: true,
		},
		{
			desc: "failed HTTP response",
			data: lfsPointer,
			options: testhelper.GitlabTestServerOptions{
				SecretToken:   secretToken,
				LfsBody:       testData,
				LfsOid:        lfsOid,
				GlRepository:  glRepository,
				LfsStatusCode: http.StatusInternalServerError,
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			c, cleanup := runTestServer(t, tc.options)
			defer cleanup()

			cfg, err := json.Marshal(c)
			require.NoError(t, err)

			env := map[string]string{
				"GL_REPOSITORY":      "project-1",
				"GL_INTERNAL_CONFIG": string(cfg),
			}

			if tc.missingEnv != "" {
				delete(env, tc.missingEnv)
			}

			cfgProvider := &mapConfig{env: env}

			var b bytes.Buffer
			reader := strings.NewReader(tc.data)

			err = smudge(&b, reader, cfgProvider)

			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.data, b.String())
			}
		})
	}
}
