package hook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type postReceiveRequest struct {
	GLRepository string   `json:"gl_repository,omitempty"`
	Identifier   string   `json:"identifier,omitempty"`
	Changes      string   `json:"changes,omitempty"`
	PushOptions  []string `json:"push_options,omitempty"`
}

// TestAllowedVerifyParams uses client cert fixtures to test TLS connections. To
// regenerate these certs, run `go generate access_test.go`.
//go:generate openssl req -newkey rsa:4096 -new -nodes -x509 -days 3650 -out testdata/certs/server.crt -keyout testdata/certs/server.key -subj "/C=US/ST=California/L=San Francisco/O=GitLab/OU=GitLab-Shell/CN=localhost" -addext "subjectAltName = IP:127.0.0.1"
func TestAccess_verifyParams(t *testing.T) {
	user, password := "user", "password"
	secretToken := "topsecret"
	glID, glRepository := "key-123", "repo-1"

	testRepo, testRepoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()
	changes := "changes1\nchanges2\nchanges3"
	protocol := "protocol"

	testRepo.GitObjectDirectory = "object/dir"
	testRepo.GitAlternateObjectDirectories = []string{"alt/object/dir1", "alt/object/dir2"}

	gitObjectDirFull := filepath.Join(testRepoPath, testRepo.GitObjectDirectory)
	var gitAlternateObjectDirsFull []string

	for _, gitAlternateObjectDirRel := range testRepo.GitAlternateObjectDirectories {
		gitAlternateObjectDirsFull = append(gitAlternateObjectDirsFull, filepath.Join(testRepoPath, gitAlternateObjectDirRel))
	}

	tempDir, cleanup := testhelper.TempDir(t)
	defer cleanup()
	testhelper.WriteShellSecretFile(t, tempDir, secretToken)

	secretFilePath := filepath.Join(tempDir, ".gitlab_shell_secret")

	serverURL, cleanup := testhelper.NewGitlabTestServer(t, testhelper.GitlabTestServerOptions{
		User:                        user,
		Password:                    password,
		SecretToken:                 secretToken,
		GLID:                        glID,
		GLRepository:                glRepository,
		Changes:                     changes,
		PostReceiveCounterDecreased: true,
		Protocol:                    protocol,
		GitPushOptions:              nil,
		GitObjectDir:                gitObjectDirFull,
		GitAlternateObjectDirs:      gitAlternateObjectDirsFull,
		RepoPath:                    testRepoPath,
		ClientCACertPath:            "testdata/certs/server.crt",
		ServerCertPath:              "testdata/certs/server.crt",
		ServerKeyPath:               "testdata/certs/server.key",
	})
	defer cleanup()

	c, err := NewGitlabAPI(config.Gitlab{
		URL:        serverURL,
		SecretFile: secretFilePath,
		HTTPSettings: config.HTTPSettings{
			User:     user,
			Password: password,
			CAFile:   "testdata/certs/server.crt",
		},
	}, config.TLS{
		CertPath: "testdata/certs/server.crt",
		KeyPath:  "testdata/certs/server.key",
	})
	require.NoError(t, err)

	badRepo := *testRepo
	badRepo.GitObjectDirectory = filepath.Join(testRepoPath, "bad/object/directory")

	testCases := []struct {
		desc                                  string
		repo                                  *gitalypb.Repository
		glRepository, glID, protocol, changes string
		allowed                               bool
	}{
		{
			desc:         "success",
			repo:         testRepo,
			glRepository: glRepository,
			glID:         glID,
			protocol:     protocol,
			changes:      changes,
			allowed:      true,
		},
		{
			desc:         "repo with bad quarantine directories",
			repo:         &badRepo,
			glRepository: glRepository,
			glID:         glID,
			protocol:     protocol,
			changes:      changes,
			allowed:      false,
		},
	}

	for _, tc := range testCases {
		allowed, _, err := c.Allowed(context.Background(), AllowedParams{
			RepoPath:                      tc.repo.RelativePath,
			GitObjectDirectory:            tc.repo.GitObjectDirectory,
			GitAlternateObjectDirectories: tc.repo.GitAlternateObjectDirectories,
			GLRepository:                  tc.glRepository,
			GLID:                          tc.glID,
			GLProtocol:                    tc.protocol,
			Changes:                       tc.changes,
		})
		require.NoError(t, err)
		require.Equal(t, tc.allowed, allowed)
	}
}

func TestAccess_escapedAndRelativeURLs(t *testing.T) {
	user, password := "user", "password"
	secretToken := "topsecret"
	glID, glRepository := "key-123", "repo-1"

	testRepo, testRepoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()
	changes := "changes1\nchanges2\nchanges3"
	protocol := "protocol"

	testRepo.GitObjectDirectory = "object/dir"
	testRepo.GitAlternateObjectDirectories = []string{"alt/object/dir1", "alt/object/dir2"}

	gitObjectDirFull := filepath.Join(testRepoPath, testRepo.GitObjectDirectory)
	var gitAlternateObjectDirsFull []string

	for _, gitAlternateObjectDirRel := range testRepo.GitAlternateObjectDirectories {
		gitAlternateObjectDirsFull = append(gitAlternateObjectDirsFull, filepath.Join(testRepoPath, gitAlternateObjectDirRel))
	}

	tempDir, cleanup := testhelper.TempDir(t)
	defer cleanup()
	testhelper.WriteShellSecretFile(t, tempDir, secretToken)

	secretFilePath := filepath.Join(tempDir, ".gitlab_shell_secret")

	testCases := []struct {
		desc            string
		escaped         bool
		relativeURLRoot string
		unixSocket      bool
	}{
		{
			desc: "unescaped URL",
		},
		{
			desc:    "escaped URL",
			escaped: true,
		},
		{
			desc:       "UNIX socket with no relative root",
			unixSocket: true,
		},
		{
			desc:            "UNIX socket with / root",
			unixSocket:      true,
			relativeURLRoot: "/",
		},
		{
			desc:            "UNIX socket with /gitlab root",
			unixSocket:      true,
			relativeURLRoot: "/gitlab",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			serverURL, cleanup := testhelper.NewGitlabTestServer(t, testhelper.GitlabTestServerOptions{
				User:                        user,
				Password:                    password,
				SecretToken:                 secretToken,
				GLID:                        glID,
				GLRepository:                glRepository,
				Changes:                     changes,
				PostReceiveCounterDecreased: true,
				Protocol:                    protocol,
				GitPushOptions:              nil,
				GitObjectDir:                gitObjectDirFull,
				GitAlternateObjectDirs:      gitAlternateObjectDirsFull,
				RepoPath:                    testRepoPath,
				RelativeURLRoot:             tc.relativeURLRoot,
				UnixSocket:                  tc.unixSocket,
			})
			defer cleanup()

			if tc.escaped {
				serverURL = url.PathEscape(serverURL)
			}

			c, err := NewGitlabAPI(config.Gitlab{
				URL:             serverURL,
				RelativeURLRoot: tc.relativeURLRoot,
				SecretFile:      secretFilePath,
				HTTPSettings: config.HTTPSettings{
					User:     user,
					Password: password,
				},
			}, config.TLS{})
			require.NoError(t, err)
			allowed, _, err := c.Allowed(context.Background(), AllowedParams{
				RepoPath:                      testRepo.RelativePath,
				GitObjectDirectory:            testRepo.GitObjectDirectory,
				GitAlternateObjectDirectories: testRepo.GitAlternateObjectDirectories,
				GLID:                          glID,
				GLRepository:                  glRepository,
				GLProtocol:                    protocol,
				Changes:                       changes,
			})
			require.NoError(t, err)
			require.True(t, allowed)
		})
	}
}

func TestAccess_allowedResponseHandling(t *testing.T) {
	testRepo, testRepoPath, cleanup := testhelper.NewTestRepo(t)

	// set git quarantine directories
	gitObjectDir := filepath.Join(testRepoPath, "quarantine", "object", "dir")
	testRepo.GitObjectDirectory = gitObjectDir
	gitAltObjectDir := filepath.Join(testRepoPath, "objects")
	testRepo.GitAlternateObjectDirectories = []string{gitAltObjectDir}

	defer cleanup()

	tempDir, cleanup := testhelper.TempDir(t)
	defer cleanup()
	testhelper.WriteShellSecretFile(t, tempDir, "secret_token")

	secretFilePath := filepath.Join(tempDir, ".gitlab_shell_secret")

	testCases := []struct {
		desc           string
		allowedHandler func(w http.ResponseWriter, r *http.Request)
		allowed        bool
		errMsg         string
	}{

		{
			desc: "allowed",
			allowedHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": true}`))
			},
			allowed: true,
			errMsg:  "",
		},

		{
			desc: "not allowed",
			allowedHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": false, "message": "this change is not allowed"}`))
			},
			allowed: false,
			errMsg:  "this change is not allowed",
		},
		{
			desc: "bad content type in response",
			allowedHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "bad mime type")
				w.WriteHeader(http.StatusOK)
			},
			allowed: false,
			errMsg:  "unsupported content type",
		},
		{
			desc: "internal server error",
			allowedHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"status": true}`))
			},
			allowed: false,
			errMsg:  "",
		},
		{
			desc: "bad response",
			allowedHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`this is not json`))
			},
			allowed: false,
			errMsg:  "decoding response from /allowed endpoint",
		},
		{
			desc: "status multiple choice",
			allowedHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMultipleChoices)
				w.Write([]byte(`{"status": true}`))
			},
			allowed: true,
			errMsg:  "",
		},
		{
			desc: "status unauthorized with message",
			allowedHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"message": "you're not allowed here'"}`))
			},
			allowed: false,
			errMsg:  "you're not allowed here",
		},
		{
			desc: "status unauthorized",
			allowedHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
			},
			allowed: false,
			errMsg:  "Internal API error",
		},
		{
			desc: "status not found",
			allowedHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"message": "not found"}`))
			},
			allowed: false,
			errMsg:  "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.allowedHandler))
			defer server.Close()

			c, err := NewGitlabAPI(config.Gitlab{
				URL:        server.URL,
				SecretFile: secretFilePath,
			}, config.TLS{})
			require.NoError(t, err)

			allowed, message, err := c.Allowed(context.Background(), AllowedParams{
				RepoPath:                      testRepo.RelativePath,
				GitObjectDirectory:            testRepo.GitObjectDirectory,
				GitAlternateObjectDirectories: testRepo.GitAlternateObjectDirectories,
				GLRepository:                  "repo-1",
				GLID:                          "key-123",
				GLProtocol:                    "http",
				Changes:                       "a\nb\nc\nd",
			})
			require.Equal(t, tc.allowed, allowed)
			if err != nil {
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.Equal(t, tc.errMsg, message)
			}
		})
	}
}

func TestAccess_preReceive(t *testing.T) {
	tempDir, cleanup := testhelper.TempDir(t)
	defer cleanup()

	testhelper.WriteShellSecretFile(t, tempDir, "secret_token")

	secretFilePath := filepath.Join(tempDir, ".gitlab_shell_secret")

	testCases := []struct {
		desc              string
		prereceiveHandler func(w http.ResponseWriter, r *http.Request)
		success           bool
		errMsg            string
	}{
		{
			desc: "everything ok",
			prereceiveHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"reference_counter_increased": true}`))
			},
			success: true,
			errMsg:  "",
		},
		{
			desc: "reference counter not increased",
			prereceiveHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"reference_counter_increased": false}`))
			},
			success: false,
			errMsg:  "",
		},
		{
			desc: "server unavailable",
			prereceiveHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"message": "server is down!"}`))
			},
			success: false,
			errMsg:  "server is down!",
		},
		{
			desc: "non json content type",
			prereceiveHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"reference_counter_increased": true}`))
			},
			success: false,
			errMsg:  "unsupported content type",
		},
		{
			desc: "bad data",
			prereceiveHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`not json`))
			},
			success: false,
			errMsg:  "decoding response from /pre_receive endpoint",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.prereceiveHandler))
			defer server.Close()

			c, err := NewGitlabAPI(config.Gitlab{
				URL:        server.URL,
				SecretFile: secretFilePath,
			}, config.TLS{})
			require.NoError(t, err)

			success, err := c.PreReceive(context.Background(), "key-123")
			require.Equal(t, tc.success, success)
			if err != nil {
				require.Contains(t, err.Error(), tc.errMsg)
			}
		})
	}
}

func TestAccess_postReceive(t *testing.T) {
	tempDir, cleanup := testhelper.TempDir(t)
	defer cleanup()

	testhelper.WriteShellSecretFile(t, tempDir, "secret_token")

	secretFilePath := filepath.Join(tempDir, ".gitlab_shell_secret")
	var receivedRequest postReceiveRequest

	testCases := []struct {
		desc               string
		postReceiveHandler func(w http.ResponseWriter, r *http.Request)
		pushOptions        []string
		success            bool
		errMsg             string
	}{
		{
			desc: "everything ok",
			postReceiveHandler: func(w http.ResponseWriter, r *http.Request) {
				err := json.NewDecoder(r.Body).Decode(&receivedRequest)
				require.NoError(t, err)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"reference_counter_decreased": true}`))
			},
			pushOptions: []string{"mr.create", "mr.label=test"},
			success:     true,
			errMsg:      "",
		},
		{
			desc: "reference counter not decreased",
			postReceiveHandler: func(w http.ResponseWriter, r *http.Request) {
				err := json.NewDecoder(r.Body).Decode(&receivedRequest)
				require.NoError(t, err)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"reference_counter_increased": false}`))
			},
			success: false,
			errMsg:  "",
		},
		{
			desc: "server unavailable",
			postReceiveHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"message": "server is down!"}`))
			},
			success: false,
			errMsg:  "server is down!",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			receivedRequest = postReceiveRequest{}
			server := httptest.NewServer(http.HandlerFunc(tc.postReceiveHandler))
			defer server.Close()

			c, err := NewGitlabAPI(config.Gitlab{
				URL:        server.URL,
				SecretFile: secretFilePath,
			}, config.TLS{})
			require.NoError(t, err)

			repositoryID := "project-123"
			identifier := "key-123"
			changes := "000 000 refs/heads/master"
			success, _, err := c.PostReceive(context.Background(), repositoryID, identifier, changes, tc.pushOptions...)
			require.Equal(t, tc.success, success)
			if err != nil {
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.Equal(t, repositoryID, receivedRequest.GLRepository)
				require.Equal(t, identifier, receivedRequest.Identifier)
				require.Equal(t, changes, receivedRequest.Changes)
				require.Equal(t, tc.pushOptions, receivedRequest.PushOptions)
			}
		})
	}
}
