package testhelper

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/helper/text"
)

// GitObjectMustExist is a test assertion that fails unless the git repo in repoPath contains sha
func GitObjectMustExist(t testing.TB, repoPath, sha string) {
	gitObjectExists(t, repoPath, sha, true)
}

// GitObjectMustNotExist is a test assertion that fails unless the git repo in repoPath contains sha
func GitObjectMustNotExist(t testing.TB, repoPath, sha string) {
	gitObjectExists(t, repoPath, sha, false)
}

func gitObjectExists(t testing.TB, repoPath, sha string, exists bool) {
	cmd := exec.Command(config.Config.Git.BinPath, "-C", repoPath, "cat-file", "-e", sha)
	cmd.Env = []string{
		"GIT_ALLOW_PROTOCOL=", // To prevent partial clone reaching remote repo over SSH
	}

	if exists {
		require.NoError(t, cmd.Run(), "checking for object should succeed")
		return
	}
	require.Error(t, cmd.Run(), "checking for object should fail")
}

// GetGitObjectDirSize gets the number of 1k blocks of a git object directory
func GetGitObjectDirSize(t testing.TB, repoPath string) int64 {
	return getGitDirSize(t, repoPath, "objects")
}

// GetGitPackfileDirSize gets the number of 1k blocks of a git object directory
func GetGitPackfileDirSize(t testing.TB, repoPath string) int64 {
	return getGitDirSize(t, repoPath, "objects", "pack")
}

func getGitDirSize(t testing.TB, repoPath string, subdirs ...string) int64 {
	cmd := exec.Command("du", "-s", "-k", filepath.Join(append([]string{repoPath}, subdirs...)...))
	output, err := cmd.Output()
	require.NoError(t, err)
	if len(output) < 2 {
		t.Error("invalid output of du -s -k")
	}

	outputSplit := strings.SplitN(string(output), "\t", 2)
	blocks, err := strconv.ParseInt(outputSplit[0], 10, 64)
	require.NoError(t, err)

	return blocks
}

func WriteBlobs(t testing.TB, testRepoPath string, n int) []string {
	var blobIDs []string
	for i := 0; i < n; i++ {
		var stdin bytes.Buffer
		stdin.Write([]byte(strconv.Itoa(time.Now().Nanosecond())))
		blobIDs = append(blobIDs, text.ChompBytes(MustRunCommand(t, &stdin, "git", "-C", testRepoPath, "hash-object", "-w", "--stdin")))
	}

	return blobIDs
}
