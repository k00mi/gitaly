package git

import (
	"context"
	"os/exec"

	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git/alternates"
	"gitlab.com/gitlab-org/gitaly/internal/git/repository"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
)

// CommandFactory knows how to properly construct different types of commands.
type CommandFactory struct {
	locator storage.Locator
	cfg     config.Cfg
}

// NewCommandFactory returns a new instance of initialized CommandFactory.
// Current implementation relies on the global var 'config.Config' and a single type of 'Locator' we currently have.
// This dependency will be removed on the next iterations in scope of: https://gitlab.com/gitlab-org/gitaly/-/issues/2699
func NewCommandFactory() *CommandFactory {
	return &CommandFactory{cfg: config.Config, locator: config.NewLocator(config.Config)}
}

func (cf *CommandFactory) gitPath() string {
	return cf.cfg.Git.BinPath
}

// unsafeCmdWithEnv creates a git.unsafeCmd with the given args, environment, and Repository
func (cf *CommandFactory) unsafeCmdWithEnv(ctx context.Context, extraEnv []string, stream CmdStream, repo repository.GitRepo, args ...string) (*command.Command, error) {
	args, env, err := argsAndEnv(repo, args...)
	if err != nil {
		return nil, err
	}

	env = append(env, extraEnv...)

	return cf.unsafeBareCmd(ctx, stream, env, args...)
}

// unsafeStdinCmd creates a git.Command with the given args and Repository that is
// suitable for Write()ing to
func (cf *CommandFactory) unsafeStdinCmd(ctx context.Context, extraEnv []string, repo repository.GitRepo, args ...string) (*command.Command, error) {
	args, env, err := argsAndEnv(repo, args...)
	if err != nil {
		return nil, err
	}

	env = append(env, extraEnv...)

	return cf.unsafeBareCmd(ctx, CmdStream{In: command.SetupStdin}, env, args...)
}

func argsAndEnv(repo repository.GitRepo, args ...string) ([]string, []string, error) {
	repoPath, env, err := alternates.PathAndEnv(repo)
	if err != nil {
		return nil, nil, err
	}

	args = append([]string{"--git-dir", repoPath}, args...)

	return args, env, nil
}

// unsafeBareCmd creates a git.Command with the given args, stdin/stdout/stderr, and env
func (cf *CommandFactory) unsafeBareCmd(ctx context.Context, stream CmdStream, env []string, args ...string) (*command.Command, error) {
	env = append(env, command.GitEnv...)

	return command.New(ctx, exec.Command(cf.gitPath(), args...), stream.In, stream.Out, stream.Err, env...)
}

// unsafeBareCmdInDir calls unsafeBareCmd in dir.
func (cf *CommandFactory) unsafeBareCmdInDir(ctx context.Context, dir string, stream CmdStream, env []string, args ...string) (*command.Command, error) {
	env = append(env, command.GitEnv...)

	cmd := exec.Command(cf.gitPath(), args...)
	cmd.Dir = dir
	return command.New(ctx, cmd, stream.In, stream.Out, stream.Err, env...)
}
