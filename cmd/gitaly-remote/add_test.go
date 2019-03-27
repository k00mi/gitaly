package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func buildGitRemote() {
	// Build the test-binary that we need
	os.Remove("gitaly-remote")
	testhelper.MustRunCommand(nil, nil, "go", "build", "-tags", "static", "gitlab.com/gitlab-org/gitaly/cmd/gitaly-remote")
}

func TestAddRemote(t *testing.T) {
	buildGitRemote()

	testCases := []struct {
		name   string
		remote string
		url    string
	}{
		{
			name:   "update-existing",
			remote: "remote",
			url:    "https://test.server.com/test.git",
		},
		{
			name:   "update-existing",
			remote: "remote",
			url:    "https://test.server.com/test.git",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
			defer cleanupFn()

			repoPath, err := helper.GetRepoPath(testRepo)
			require.NoError(t, err)

			cmd := exec.Command("./gitaly-remote", repoPath, tc.remote)
			cmd.Stdin = strings.NewReader(tc.url)
			err = cmd.Run()
			require.NoError(t, err)

			out := testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "remote", "get-url", tc.remote)
			require.Equal(t, tc.url, strings.TrimSpace(string(out)))
		})
	}

}
