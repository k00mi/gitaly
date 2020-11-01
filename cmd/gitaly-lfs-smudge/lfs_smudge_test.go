package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
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

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	env := map[string]string{
		"GL_REPOSITORY":      "project-1",
		"GL_INTERNAL_CONFIG": string(cfg),
		"GITALY_LOG_DIR":     tmpDir,
	}
	cfgProvider := &mapConfig{env: env}
	initLogging(cfgProvider)

	err = smudge(&b, reader, cfgProvider)
	require.NoError(t, err)
	require.Equal(t, testData, b.String())

	logFilename := filepath.Join(tmpDir, "gitaly_lfs_smudge.log")
	require.FileExists(t, logFilename)

	data, err := ioutil.ReadFile(logFilename)
	require.NoError(t, err)
	d := string(data)

	require.Contains(t, d, `"msg":"Finished HTTP request"`)
	require.Contains(t, d, `"status":200`)
	require.Contains(t, d, `"gl_repository":"project-1"`)
	require.Contains(t, d, `"oid":"`+lfsOid)
}

func TestUnsuccessfulLfsSmudge(t *testing.T) {
	testCases := []struct {
		desc               string
		data               string
		missingEnv         string
		expectedError      bool
		options            testhelper.GitlabTestServerOptions
		expectedLogMessage string
	}{
		{
			desc:          "bad LFS pointer",
			data:          "test data",
			options:       defaultOptions,
			expectedError: false,
		},
		{
			desc:               "missing GL_REPOSITORY",
			data:               lfsPointer,
			missingEnv:         "GL_REPOSITORY",
			options:            defaultOptions,
			expectedError:      true,
			expectedLogMessage: "GL_REPOSITORY is not defined",
		},
		{
			desc:               "missing GL_INTERNAL_CONFIG",
			data:               lfsPointer,
			missingEnv:         "GL_INTERNAL_CONFIG",
			options:            defaultOptions,
			expectedError:      true,
			expectedLogMessage: "unable to retrieve GL_INTERNAL_CONFIG",
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
			expectedError:      true,
			expectedLogMessage: "error loading LFS object",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			c, cleanup := runTestServer(t, tc.options)
			defer cleanup()

			cfg, err := json.Marshal(c)
			require.NoError(t, err)

			tmpDir, err := ioutil.TempDir("", "")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			env := map[string]string{
				"GL_REPOSITORY":      "project-1",
				"GL_INTERNAL_CONFIG": string(cfg),
				"GITALY_LOG_DIR":     tmpDir,
			}

			if tc.missingEnv != "" {
				delete(env, tc.missingEnv)
			}

			cfgProvider := &mapConfig{env: env}

			var b bytes.Buffer
			reader := strings.NewReader(tc.data)

			initLogging(cfgProvider)
			err = smudge(&b, reader, cfgProvider)

			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.data, b.String())
			}

			logFilename := filepath.Join(tmpDir, "gitaly_lfs_smudge.log")
			require.FileExists(t, logFilename)

			data, err := ioutil.ReadFile(logFilename)
			require.NoError(t, err)

			if tc.expectedLogMessage != "" {
				require.Contains(t, string(data), tc.expectedLogMessage)
			}
		})
	}
}
