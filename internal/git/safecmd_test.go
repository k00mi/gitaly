package git

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/git/hooks"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/helper/text"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func TestFlagValidation(t *testing.T) {
	for _, tt := range []struct {
		option Option
		valid  bool
	}{
		// valid Flag inputs
		{option: Flag{Name: "-k"}, valid: true},
		{option: Flag{Name: "-K"}, valid: true},
		{option: Flag{Name: "--asdf"}, valid: true},
		{option: Flag{Name: "--asdf-qwer"}, valid: true},
		{option: Flag{Name: "--asdf=qwerty"}, valid: true},
		{option: Flag{Name: "-D=A"}, valid: true},
		{option: Flag{Name: "-D="}, valid: true},

		// valid ValueFlag inputs
		{option: ValueFlag{"-k", "adsf"}, valid: true},
		{option: ValueFlag{"-k", "--anything"}, valid: true},
		{option: ValueFlag{"-k", ""}, valid: true},

		// valid SubSubCmd inputs
		{option: SubSubCmd{"meow"}, valid: true},

		// valid ConfigPair inputs
		{option: ConfigPair{Key: "a.b.c", Value: "d"}, valid: true},
		{option: ConfigPair{Key: "core.sound", Value: "meow"}, valid: true},
		{option: ConfigPair{Key: "asdf-qwer.1234-5678", Value: ""}, valid: true},
		{option: ConfigPair{Key: "http.https://user@example.com/repo.git.user", Value: "kitty"}, valid: true},

		// invalid Flag inputs
		{option: Flag{Name: "-*"}},          // invalid character
		{option: Flag{Name: "a"}},           // missing dash
		{option: Flag{Name: "[["}},          // suspicious characters
		{option: Flag{Name: "||"}},          // suspicious characters
		{option: Flag{Name: "asdf=qwerty"}}, // missing dash

		// invalid ValueFlag inputs
		{option: ValueFlag{"k", "asdf"}}, // missing dash

		// invalid SubSubCmd inputs
		{option: SubSubCmd{"--meow"}}, // cannot start with dash

		// invalid ConfigPair inputs
		{option: ConfigPair{Key: "", Value: ""}},            // key cannot be empty
		{option: ConfigPair{Key: " ", Value: ""}},           // key cannot be whitespace
		{option: ConfigPair{Key: "asdf", Value: ""}},        // two components required
		{option: ConfigPair{Key: "asdf.", Value: ""}},       // 2nd component must be non-empty
		{option: ConfigPair{Key: "--asdf.asdf", Value: ""}}, // key cannot start with dash
		{option: ConfigPair{Key: "as[[df.asdf", Value: ""}}, // 1st component cannot contain non-alphanumeric
		{option: ConfigPair{Key: "asdf.as]]df", Value: ""}}, // 2nd component cannot contain non-alphanumeric
	} {
		args, err := tt.option.ValidateArgs()
		if tt.valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err,
				"expected error, but args %v passed validation", args)
			require.True(t, IsInvalidArgErr(err))
		}
	}
}

func TestSafeCmdInvalidArg(t *testing.T) {
	for _, tt := range []struct {
		globals []Option
		subCmd  SubCmd
		errMsg  string
	}{
		{
			subCmd: SubCmd{Name: "--meow"},
			errMsg: `invalid sub command name "--meow": invalid argument`,
		},
		{
			subCmd: SubCmd{
				Name:  "update-ref",
				Flags: []Option{Flag{Name: "woof"}},
			},
			errMsg: `flag "woof" failed regex validation: invalid argument`,
		},
		{
			subCmd: SubCmd{
				Name: "update-ref",
				Args: []string{"--tweet"},
			},
			errMsg: `positional arg "--tweet" cannot start with dash '-': invalid argument`,
		},
		{
			subCmd: SubCmd{
				Name:  "update-ref",
				Flags: []Option{SubSubCmd{"-invalid"}},
			},
			errMsg: `invalid sub-sub command name "-invalid": invalid argument`,
		},
	} {
		_, err := SafeCmd(
			context.Background(),
			&gitalypb.Repository{},
			tt.globals,
			tt.subCmd,
			WithRefTxHook(context.Background(), &gitalypb.Repository{}, config.Config),
		)
		require.EqualError(t, err, tt.errMsg)
		require.True(t, IsInvalidArgErr(err))
	}
}

func TestSafeCmdValid(t *testing.T) {
	testRepo, testRepoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	ctx, cancel := testhelper.Context()
	defer cancel()

	reenableGitCmd := disableGitCmd()
	defer reenableGitCmd()

	hooksPath := "core.hooksPath=" + hooks.Path(config.Config)

	endOfOptions := "--end-of-options"

	for _, tt := range []struct {
		desc       string
		globals    []Option
		subCmd     SubCmd
		expectArgs []string
	}{
		{
			desc:       "no args",
			subCmd:     SubCmd{Name: "update-ref"},
			expectArgs: []string{"-c", hooksPath, "update-ref", endOfOptions},
		},
		{
			desc: "single option",
			globals: []Option{
				Flag{Name: "--aaaa-bbbb"},
			},
			subCmd:     SubCmd{Name: "update-ref"},
			expectArgs: []string{"--aaaa-bbbb", "-c", hooksPath, "update-ref", endOfOptions},
		},
		{
			desc: "empty arg and postsep args",
			subCmd: SubCmd{
				Name:        "update-ref",
				Args:        []string{""},
				PostSepArgs: []string{"-woof", ""},
			},
			expectArgs: []string{"-c", hooksPath, "update-ref", "", endOfOptions, "--", "-woof", ""},
		},
		{
			desc: "full blown",
			globals: []Option{
				Flag{Name: "-a"},
				ValueFlag{"-b", "c"},
			},
			subCmd: SubCmd{
				Name: "update-ref",
				Flags: []Option{
					Flag{Name: "-e"},
					ValueFlag{"-f", "g"},
					Flag{Name: "-h=i"},
				},
				Args:        []string{"1", "2"},
				PostSepArgs: []string{"3", "4", "5"},
			},
			expectArgs: []string{"-a", "-b", "c", "-c", hooksPath, "update-ref", "-e", "-f", "g", "-h=i", "1", "2", endOfOptions, "--", "3", "4", "5"},
		},
		{
			desc: "output to stdout",
			subCmd: SubCmd{
				Name: "update-ref",
				Flags: []Option{
					SubSubCmd{"verb"},
					OutputToStdout,
					Flag{Name: "--adjective"},
				},
			},
			expectArgs: []string{"-c", hooksPath, "update-ref", "verb", "-", "--adjective", endOfOptions},
		},
		{
			desc: "multiple value flags",
			globals: []Option{
				Flag{Name: "--contributing"},
				ValueFlag{"--author", "a-gopher"},
			},
			subCmd: SubCmd{
				Name: "update-ref",
				Args: []string{"mr"},
				Flags: []Option{
					Flag{Name: "--is-important"},
					ValueFlag{"--why", "looking-for-first-contribution"},
				},
			},
			expectArgs: []string{"--contributing", "--author", "a-gopher", "-c", hooksPath, "update-ref", "--is-important", "--why", "looking-for-first-contribution", "mr", endOfOptions},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			opts := []CmdOpt{WithRefTxHook(ctx, &gitalypb.Repository{}, config.Config)}

			cmd, err := SafeCmd(ctx, testRepo, tt.globals, tt.subCmd, opts...)
			require.NoError(t, err)
			// ignore first 3 indeterministic args (executable path and repo args)
			require.Equal(t, tt.expectArgs, cmd.Args()[3:])

			cmd, err = SafeCmdWithEnv(ctx, nil, testRepo, tt.globals, tt.subCmd, opts...)
			require.NoError(t, err)
			// ignore first 3 indeterministic args (executable path and repo args)
			require.Equal(t, tt.expectArgs, cmd.Args()[3:])

			cmd, err = SafeStdinCmd(ctx, testRepo, tt.globals, tt.subCmd, opts...)
			require.NoError(t, err)
			require.Equal(t, tt.expectArgs, cmd.Args()[3:])

			cmd, err = SafeBareCmd(ctx, CmdStream{}, nil, tt.globals, tt.subCmd, opts...)
			require.NoError(t, err)
			// ignore first indeterministic arg (executable path)
			require.Equal(t, tt.expectArgs, cmd.Args()[1:])

			cmd, err = SafeCmdWithoutRepo(ctx, tt.globals, tt.subCmd, opts...)
			require.NoError(t, err)
			require.Equal(t, tt.expectArgs, cmd.Args()[1:])

			cmd, err = SafeBareCmdInDir(ctx, testRepoPath, CmdStream{}, nil, tt.globals, tt.subCmd, opts...)
			require.NoError(t, err)
			require.Equal(t, tt.expectArgs, cmd.Args()[1:])
		})
	}
}

func TestSafeCmdWithEnv(t *testing.T) {
	testRepo, _, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	ctx, cancel := testhelper.Context()
	defer cancel()

	reenableGitCmd := disableGitCmd()
	defer reenableGitCmd()

	globals := []Option{
		Flag{Name: "--aaaa-bbbb"},
	}

	subCmd := SubCmd{Name: "update-ref"}
	endOfOptions := "--end-of-options"
	expectArgs := []string{"--aaaa-bbbb", "-c", "core.hooksPath=" + hooks.Path(config.Config), "update-ref", endOfOptions}

	env := []string{"TEST_VAR1=1", "TEST_VAR2=2"}

	opts := []CmdOpt{WithRefTxHook(ctx, &gitalypb.Repository{}, config.Config)}
	cmd, err := SafeCmdWithEnv(ctx, env, testRepo, globals, subCmd, opts...)
	require.NoError(t, err)
	// ignore first 3 indeterministic args (executable path and repo args)
	require.Equal(t, expectArgs, cmd.Args()[3:])
	require.Subset(t, cmd.Env(), env)
}

func disableGitCmd() testhelper.Cleanup {
	oldBinPath := config.Config.Git.BinPath
	config.Config.Git.BinPath = "/bin/echo"
	return func() { config.Config.Git.BinPath = oldBinPath }
}

func TestSafeBareCmdInDir(t *testing.T) {
	t.Run("no dir specified", func(t *testing.T) {
		ctx, cancel := testhelper.Context()
		defer cancel()

		_, err := SafeBareCmdInDir(ctx, "", CmdStream{}, nil, nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no 'dir' provided")
	})

	t.Run("runs in dir", func(t *testing.T) {
		_, repoPath, cleanup := testhelper.NewTestRepoWithWorktree(t)
		defer cleanup()

		ctx, cancel := testhelper.Context()
		defer cancel()

		var stderr bytes.Buffer
		cmd, err := SafeBareCmdInDir(ctx, repoPath, CmdStream{Err: &stderr}, nil, nil, SubCmd{
			Name: "rev-parse",
			Args: []string{"master"},
		})
		require.NoError(t, err)

		revData, err := ioutil.ReadAll(cmd)
		require.NoError(t, err)

		require.NoError(t, cmd.Wait(), stderr.String())

		require.Equal(t, "1e292f8fedd741b75372e19097c76d327140c312", text.ChompBytes(revData))
	})

	t.Run("doesn't runs in non existing dir", func(t *testing.T) {
		ctx, cancel := testhelper.Context()
		defer cancel()

		var stderr bytes.Buffer
		_, err := SafeBareCmdInDir(ctx, "non-existing-dir", CmdStream{Err: &stderr}, nil, nil, SubCmd{
			Name: "rev-parse",
			Args: []string{"master"},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "no such file or directory")
	})
}

func TestSafeCmd(t *testing.T) {
	t.Run("stderr is empty if no error", func(t *testing.T) {
		repo, _, cleanup := testhelper.NewTestRepoWithWorktree(t)
		defer cleanup()

		ctx, cancel := testhelper.Context()
		defer cancel()

		var stderr bytes.Buffer
		cmd, err := SafeCmd(ctx, repo, nil,
			SubCmd{
				Name: "rev-parse",
				Args: []string{"master"},
			},
			WithStderr(&stderr),
		)
		require.NoError(t, err)

		revData, err := ioutil.ReadAll(cmd)
		require.NoError(t, err)

		require.NoError(t, cmd.Wait(), stderr.String())

		require.Equal(t, "1e292f8fedd741b75372e19097c76d327140c312", text.ChompBytes(revData))
		require.Empty(t, stderr.String())
	})

	t.Run("stderr contains error message on failure", func(t *testing.T) {
		repo, _, cleanup := testhelper.NewTestRepoWithWorktree(t)
		defer cleanup()

		ctx, cancel := testhelper.Context()
		defer cancel()

		var stderr bytes.Buffer
		cmd, err := SafeCmd(ctx, repo, nil, SubCmd{
			Name: "rev-parse",
			Args: []string{"invalid-ref"},
		},
			WithStderr(&stderr),
		)
		require.NoError(t, err)

		revData, err := ioutil.ReadAll(cmd)
		require.NoError(t, err)

		require.Error(t, cmd.Wait())

		require.Equal(t, "invalid-ref", text.ChompBytes(revData))
		require.Contains(t, stderr.String(), "fatal: ambiguous argument 'invalid-ref': unknown revision or path not in the working tree.")
	})
}
