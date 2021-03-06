package testhelper

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
)

// EnableGitProtocolV2Support replaces the git binary in config with a
// wrapper that allows the protocol to be tested. It returns a function that
// restores the given settings as well as an array of environment variables
// which need to be set when invoking Git with this setup.
func EnableGitProtocolV2Support(t *testing.T) func() {
	envPath := filepath.Join(testDirectory, "git-env")

	script := fmt.Sprintf(`#!/bin/sh
env | grep ^GIT_PROTOCOL= >>"%s"
exec "%s" "$@"
`, envPath, config.Config.Git.BinPath)

	dir, cleanupDir := TempDir(t)

	path := filepath.Join(dir, "git")
	cleanupExe, err := WriteExecutable(path, []byte(script))
	if !assert.NoError(t, err) {
		cleanupDir()
		t.FailNow()
	}

	oldGitBinPath := config.Config.Git.BinPath
	config.Config.Git.BinPath = path
	return func() {
		os.Remove(envPath)
		config.Config.Git.BinPath = oldGitBinPath
		cleanupExe()
		cleanupDir()
	}
}
