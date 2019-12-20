package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitlabshell"
	gitalylog "gitlab.com/gitlab-org/gitaly/internal/log"
)

func main() {
	var logger = gitalylog.NewHookLogger()

	if len(os.Args) < 2 {
		logger.Fatal(errors.New("requires hook name"))
	}

	subCmd := os.Args[1]

	if subCmd == "check" {
		configPath := os.Args[2]

		status, err := check(configPath)
		if err != nil {
			log.Fatal(err)
		}

		os.Exit(status)
	}

	gitalyRubyDir := os.Getenv("GITALY_RUBY_DIR")
	if gitalyRubyDir == "" {
		logger.Fatal(errors.New("GITALY_RUBY_DIR not set"))
	}

	rubyHookPath := filepath.Join(gitalyRubyDir, "gitlab-shell", "hooks", subCmd)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var hookCmd *exec.Cmd

	switch subCmd {
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

	cmd, err := command.New(ctx, hookCmd, os.Stdin, os.Stdout, os.Stderr, os.Environ()...)
	if err != nil {
		logger.Fatalf("error when starting command for %v: %v", rubyHookPath, err)
	}

	if err = cmd.Wait(); err != nil {
		os.Exit(1)
	}
}

// GitlabShellConfig contains a subset of gitlabshell's config.yml
type GitlabShellConfig struct {
	GitlabURL    string       `yaml:"gitlab_url"`
	HTTPSettings HTTPSettings `yaml:"http_settings"`
}

// HTTPSettings contains fields for http settings
type HTTPSettings struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func check(configPath string) (int, error) {
	cfgFile, err := os.Open(configPath)
	if err != nil {
		return 1, fmt.Errorf("error when opening config file: %v", err)
	}
	defer cfgFile.Close()

	var c config.Cfg

	if _, err := toml.DecodeReader(cfgFile, &c); err != nil {
		fmt.Println(err)
		return 1, err
	}

	cmd := exec.Command(filepath.Join(c.GitlabShell.Dir, "bin", "check"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), gitlabshell.EnvFromConfig(c)...)

	if err = cmd.Run(); err != nil {
		if status, ok := command.ExitStatus(err); ok {
			return status, nil
		}
		return 1, err
	}

	return 0, nil
}
