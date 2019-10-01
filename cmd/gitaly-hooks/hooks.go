package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/log"
)

func main() {
	var logger = log.NewHookLogger()

	if len(os.Args) < 2 {
		logger.Fatal(errors.New("requires hook name"))
	}

	gitlabRubyDir := os.Getenv("GITALY_RUBY_DIR")
	if gitlabRubyDir == "" {
		logger.Fatal(errors.New("GITALY_RUBY_DIR not set"))
	}

	hookName := os.Args[1]
	rubyHookPath := filepath.Join(gitlabRubyDir, "gitlab-shell", "hooks", hookName)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var hookCmd *exec.Cmd

	switch hookName {
	case "update":
		args := os.Args[2:]
		if len(args) != 3 {
			logger.Fatal(errors.New("update hook missing required arguments"))
		}

		hookCmd = exec.Command(rubyHookPath, args...)
	case "pre-receive", "post-receive":
		hookCmd = exec.Command(rubyHookPath)
	default:
		logger.Fatal(errors.New("hook name invalid"))
	}

	var stderr bytes.Buffer
	mw := io.MultiWriter(&stderr, os.Stderr)

	cmd, err := command.New(ctx, hookCmd, os.Stdin, os.Stdout, mw, os.Environ()...)
	if err != nil {
		logger.Fatalf("error when starting command for %v: %v", rubyHookPath, err)
	}

	if err = cmd.Wait(); err != nil {
		os.Exit(1)
	}
}
