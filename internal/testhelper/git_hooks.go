package testhelper

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/git/hooks"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
)

// dump the env vars that the custom hooks receives to a file
func WriteEnvToCustomHook(t testing.TB, repoPath, hookName string) (string, func()) {
	hookOutputTemp, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	require.NoError(t, hookOutputTemp.Close())

	hookContent := fmt.Sprintf("#!/bin/sh\n/usr/bin/env > %s\n", hookOutputTemp.Name())

	cleanupCustomHook, err := WriteCustomHook(repoPath, hookName, []byte(hookContent))
	require.NoError(t, err)

	return hookOutputTemp.Name(), func() {
		cleanupCustomHook()
		os.Remove(hookOutputTemp.Name())
	}
}

// WriteCheckNewObjectExistsHook writes a pre-receive hook which only succeeds
// if it can find the object in the quarantine directory. if
// GIT_OBJECT_DIRECTORY and GIT_ALTERNATE_OBJECT_DIRECTORIES were not passed
// through correctly to the hooks, it will fail
func WriteCheckNewObjectExistsHook(t *testing.T, repoPath string) func() {
	hook := fmt.Sprintf(`#!/usr/bin/env ruby
STDIN.each_line do |line|
  new_object = line.split(' ')[1]
  exit 1 unless new_object
  exit 1 unless	system(*%%W[%s cat-file -e #{new_object}])
end
`, config.Config.Git.BinPath)

	cleanup, err := WriteCustomHook(repoPath, "pre-receive", []byte(hook))
	require.NoError(t, err)

	return cleanup
}

// write a hook in the repo/path.git/custom_hooks directory
func WriteCustomHook(repoPath, name string, content []byte) (func(), error) {
	fullPath := filepath.Join(repoPath, "custom_hooks", name)
	return WriteExecutable(fullPath, content)
}

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

// GetGitEnvData reads and returns the content of testGitEnv
func GetGitEnvData() (string, error) {
	gitEnvBytes, err := ioutil.ReadFile(filepath.Join(testDirectory, "git-env"))

	if err != nil {
		return "", err
	}

	return string(gitEnvBytes), nil
}
