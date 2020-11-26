package main

import (
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

var gitalySSHPath string

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	defer testhelper.MustHaveNoChildProcess()

	cleanup := testhelper.Configure()
	defer cleanup()

	testhelper.ConfigureGitalySSH()
	testhelper.ConfigureGitalyHooksBinary()

	gitalySSHPath = filepath.Join(config.Config.BinDir, "gitaly-ssh")

	return m.Run()
}
