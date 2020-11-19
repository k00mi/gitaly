package hook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/metadata"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func TestPrereceive_customHooks(t *testing.T) {
	repo, repoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	hookManager := NewManager(GitlabAPIStub, config.Config)

	standardEnv := []string{
		fmt.Sprintf("GITALY_SOCKET=%s", config.Config.GitalyInternalSocketPath()),
		"GITALY_TOKEN=secret",
		"GL_ID=1234",
		fmt.Sprintf("GL_PROJECT_PATH=%s", repo.GetGlProjectPath()),
		"GL_PROTOCOL=web",
		fmt.Sprintf("GL_REPO=%s", repo),
		fmt.Sprintf("GL_REPOSITORY=%s", repo.GetGlRepository()),
		"GL_USERNAME=user",
	}

	primaryEnv, err := metadata.Transaction{
		ID: 1234, Node: "primary", Primary: true,
	}.Env()
	require.NoError(t, err)

	secondaryEnv, err := metadata.Transaction{
		ID: 1234, Node: "secondary", Primary: false,
	}.Env()
	require.NoError(t, err)

	testCases := []struct {
		desc           string
		env            []string
		hook           string
		stdin          string
		expectedErr    string
		expectedStdout string
		expectedStderr string
	}{
		{
			desc:           "hook receives environment variables",
			env:            standardEnv,
			hook:           "#!/bin/sh\nenv | grep -e '^GL_' -e '^GITALY_' | sort\n",
			expectedStdout: strings.Join(standardEnv, "\n") + "\n",
		},
		{
			desc:           "hook can write to stderr and stdout",
			env:            standardEnv,
			hook:           "#!/bin/sh\necho foo >&1 && echo bar >&2\n",
			expectedStdout: "foo\n",
			expectedStderr: "bar\n",
		},
		{
			desc:           "hook receives standard input",
			env:            standardEnv,
			hook:           "#!/bin/sh\ncat\n",
			stdin:          "foo\n",
			expectedStdout: "foo\n",
		},
		{
			desc:           "hook succeeds without consuming stdin",
			env:            standardEnv,
			hook:           "#!/bin/sh\necho foo\n",
			stdin:          "ignore me\n",
			expectedStdout: "foo\n",
		},
		{
			desc:        "invalid hook results in error",
			env:         standardEnv,
			hook:        "",
			expectedErr: "exec format error",
		},
		{
			desc:        "failing hook results in error",
			env:         standardEnv,
			hook:        "#!/bin/sh\nexit 123",
			expectedErr: "exit status 123",
		},
		{
			desc:           "hook is executed on primary",
			env:            append(standardEnv, primaryEnv),
			hook:           "#!/bin/sh\necho foo\n",
			expectedStdout: "foo\n",
		},
		{
			desc: "hook is not executed on secondary",
			env:  append(standardEnv, secondaryEnv),
			hook: "#!/bin/sh\necho foo\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cleanup := testhelper.Context()
			defer cleanup()

			cleanup, err := testhelper.WriteCustomHook(repoPath, "pre-receive", []byte(tc.hook))
			require.NoError(t, err)
			defer cleanup()

			var stdout, stderr bytes.Buffer
			err = hookManager.PreReceiveHook(ctx, repo, tc.env, strings.NewReader(tc.stdin), &stdout, &stderr)

			if tc.expectedErr != "" {
				require.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.expectedStdout, stdout.String())
			require.Equal(t, tc.expectedStderr, stderr.String())
		})
	}
}

type prereceiveAPIMock struct {
	allowed    func(context.Context, *gitalypb.Repository, string, string, string, string) (bool, string, error)
	prereceive func(context.Context, string) (bool, error)
}

func (m *prereceiveAPIMock) Allowed(ctx context.Context, repo *gitalypb.Repository, glRepository, glID, glProtocol, changes string) (bool, string, error) {
	return m.allowed(ctx, repo, glRepository, glID, glProtocol, changes)
}

func (m *prereceiveAPIMock) PreReceive(ctx context.Context, glRepository string) (bool, error) {
	return m.prereceive(ctx, glRepository)
}

func (m *prereceiveAPIMock) Check(ctx context.Context) (*CheckInfo, error) {
	return nil, errors.New("unexpected call")
}

func (m *prereceiveAPIMock) PostReceive(context.Context, string, string, string, ...string) (bool, []PostReceiveMessage, error) {
	return true, nil, errors.New("unexpected call")
}

func TestPrereceive_gitlab(t *testing.T) {
	testRepo, testRepoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	standardEnv := []string{
		fmt.Sprintf("GITALY_SOCKET=%s", config.Config.GitalyInternalSocketPath()),
		"GITALY_TOKEN=secret",
		"GL_ID=1234",
		fmt.Sprintf("GL_PROJECT_PATH=%s", testRepo.GetGlProjectPath()),
		"GL_PROTOCOL=web",
		fmt.Sprintf("GL_REPO=%s", testRepo),
		fmt.Sprintf("GL_REPOSITORY=%s", testRepo.GetGlRepository()),
		"GL_USERNAME=user",
	}

	testCases := []struct {
		desc           string
		env            []string
		changes        string
		allowed        func(*testing.T, context.Context, *gitalypb.Repository, string, string, string, string) (bool, string, error)
		prereceive     func(*testing.T, context.Context, string) (bool, error)
		expectHookCall bool
		expectedErr    error
	}{
		{
			desc:    "allowed change",
			env:     standardEnv,
			changes: "changes\n",
			allowed: func(t *testing.T, ctx context.Context, repo *gitalypb.Repository, glRepo, glID, glProtocol, changes string) (bool, string, error) {
				require.Equal(t, testRepo, repo)
				require.Equal(t, testRepo.GlRepository, glRepo)
				require.Equal(t, "1234", glID)
				require.Equal(t, "web", glProtocol)
				require.Equal(t, "changes\n", changes)
				return true, "", nil
			},
			prereceive: func(t *testing.T, ctx context.Context, glRepo string) (bool, error) {
				require.Equal(t, testRepo.GlRepository, glRepo)
				return true, nil
			},
			expectHookCall: true,
		},
		{
			desc: "disallowed change",
			env:  standardEnv,
			allowed: func(t *testing.T, ctx context.Context, repo *gitalypb.Repository, glRepo, glID, glProtocol, changes string) (bool, string, error) {
				return false, "you shall not pass", nil
			},
			expectHookCall: false,
			expectedErr:    errors.New("you shall not pass"),
		},
		{
			desc: "allowed returns error",
			env:  standardEnv,
			allowed: func(t *testing.T, ctx context.Context, repo *gitalypb.Repository, glRepo, glID, glProtocol, changes string) (bool, string, error) {
				return false, "", errors.New("oops")
			},
			expectHookCall: false,
			expectedErr:    errors.New("GitLab: oops"),
		},
		{
			desc: "prereceive rejects",
			env:  standardEnv,
			allowed: func(t *testing.T, ctx context.Context, repo *gitalypb.Repository, glRepo, glID, glProtocol, changes string) (bool, string, error) {
				return true, "", nil
			},
			prereceive: func(t *testing.T, ctx context.Context, glRepo string) (bool, error) {
				return false, nil
			},
			expectHookCall: true,
			expectedErr:    errors.New(""),
		},
		{
			desc: "prereceive errors",
			env:  standardEnv,
			allowed: func(t *testing.T, ctx context.Context, repo *gitalypb.Repository, glRepo, glID, glProtocol, changes string) (bool, string, error) {
				return true, "", nil
			},
			prereceive: func(t *testing.T, ctx context.Context, glRepo string) (bool, error) {
				return false, errors.New("prereceive oops")
			},
			expectHookCall: true,
			expectedErr:    helper.ErrInternalf("calling pre_receive endpoint: %v", errors.New("prereceive oops")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cleanup := testhelper.Context()
			defer cleanup()

			gitlabAPI := prereceiveAPIMock{
				allowed: func(ctx context.Context, repo *gitalypb.Repository, glRepo, glID, glProtocol, changes string) (bool, string, error) {
					return tc.allowed(t, ctx, repo, glRepo, glID, glProtocol, changes)
				},
				prereceive: func(ctx context.Context, glRepo string) (bool, error) {
					return tc.prereceive(t, ctx, glRepo)
				},
			}

			hookManager := NewManager(&gitlabAPI, config.Config)

			cleanup, err := testhelper.WriteCustomHook(testRepoPath, "pre-receive", []byte("#!/bin/sh\necho called\n"))
			require.NoError(t, err)
			defer cleanup()

			var stdout, stderr bytes.Buffer
			err = hookManager.PreReceiveHook(ctx, testRepo, tc.env, strings.NewReader(tc.changes), &stdout, &stderr)

			if tc.expectedErr != nil {
				require.Equal(t, tc.expectedErr, err)
			} else {
				require.NoError(t, err)
			}

			if tc.expectHookCall {
				require.Equal(t, "called\n", stdout.String())
			} else {
				require.Empty(t, stdout.String())
			}
			require.Empty(t, stderr.String())
		})
	}
}
