package testhelper

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/gitaly/internal/git/hooks"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
)

// CaptureHookEnv creates a bogus 'update' Git hook to sniff out what
// environment variables get set for hooks.
func CaptureHookEnv(t testing.TB) (string, func()) {
	tempDir, cleanup := TempDir(t)

	oldOverride := hooks.Override
	hooks.Override = filepath.Join(tempDir, "hooks")
	hookOutputFile := filepath.Join(tempDir, "hook.env")

	if !assert.NoError(t, os.MkdirAll(hooks.Override, 0755)) {
		cleanup()
		t.FailNow()
	}

	script := []byte(`
#!/bin/sh
env | grep -e ^GIT -e ^GL_ > ` + hookOutputFile + "\n")

	if !assert.NoError(t, ioutil.WriteFile(filepath.Join(hooks.Override, "update"), script, 0755)) {
		cleanup()
		t.FailNow()
	}

	return hookOutputFile, func() {
		cleanup()
		hooks.Override = oldOverride
	}
}

// ConfigureGitalyHooksBinary builds gitaly-hooks command for tests
func ConfigureGitalyHooksBinary() {
	if config.Config.BinDir == "" {
		log.Fatal("config.Config.BinDir must be set")
	}

	goBuildArgs := []string{
		"build",
		"-o",
		filepath.Join(config.Config.BinDir, "gitaly-hooks"),
		"gitlab.com/gitlab-org/gitaly/cmd/gitaly-hooks",
	}
	MustRunCommand(nil, nil, "go", goBuildArgs...)
}
