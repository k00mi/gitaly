package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git/repository"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

var (
	invalidationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitaly_invalid_commands_total",
			Help: "Total number of invalid arguments tried to execute",
		},
		[]string{"command"},
	)

	// ErrInvalidArg represent family of errors to report about bad argument used to make a call.
	ErrInvalidArg = errors.New("invalid argument")
)

func init() {
	prometheus.MustRegister(invalidationTotal)
}

func incrInvalidArg(subcmdName string) {
	invalidationTotal.WithLabelValues(subcmdName).Inc()
}

// Cmd is an interface for safe git commands
type Cmd interface {
	CommandArgs() ([]string, error)
	Subcommand() string
}

// SubCmd represents a specific git command
type SubCmd struct {
	Name        string   // e.g. "log", or "cat-file", or "worktree"
	Flags       []Option // optional flags before the positional args
	Args        []string // positional args after all flags
	PostSepArgs []string // post separator (i.e. "--") positional args
}

// cmdStream represents standard input/output streams for a command
type cmdStream struct {
	In       io.Reader // standard input
	Out, Err io.Writer // standard output and error
}

// Subcommand returns the subcommand name
func (sc SubCmd) Subcommand() string { return sc.Name }

// CommandArgs checks all arguments in the sub command and validates them
func (sc SubCmd) CommandArgs() ([]string, error) {
	var safeArgs []string

	if _, ok := subcommands[sc.Name]; !ok {
		return nil, fmt.Errorf("invalid sub command name %q: %w", sc.Name, ErrInvalidArg)
	}
	safeArgs = append(safeArgs, sc.Name)

	commandArgs, err := assembleCommandArgs(sc.Name, sc.Flags, sc.Args, sc.PostSepArgs)
	if err != nil {
		return nil, err
	}
	safeArgs = append(safeArgs, commandArgs...)

	return safeArgs, nil
}

func assembleCommandArgs(command string, flags []Option, args []string, postSepArgs []string) ([]string, error) {
	var commandArgs []string

	for _, o := range flags {
		args, err := o.OptionArgs()
		if err != nil {
			return nil, err
		}
		commandArgs = append(commandArgs, args...)
	}

	for _, a := range args {
		if err := validatePositionalArg(a); err != nil {
			return nil, err
		}
		commandArgs = append(commandArgs, a)
	}

	if supportsEndOfOptions(command) {
		commandArgs = append(commandArgs, "--end-of-options")
	}

	if len(postSepArgs) > 0 {
		commandArgs = append(commandArgs, "--")
	}

	// post separator args do not need any validation
	commandArgs = append(commandArgs, postSepArgs...)

	return commandArgs, nil
}

// GlobalOption is an interface for all options which can be globally applied
// to git commands. This is the command-inspecific part before the actual
// command that's being run, e.g. the `-c` part in `git -c foo.bar=value
// command`.
type GlobalOption interface {
	GlobalArgs() ([]string, error)
}

// Option is a git command line flag with validation logic
type Option interface {
	OptionArgs() ([]string, error)
}

// SubSubCmd is a positional argument that appears in the list of options for
// a subcommand.
type SubSubCmd struct {
	// Name is the name of the subcommand, e.g. "remote" in `git remote set-url`
	Name string
	// Action is the action of the subcommand, e.g. "set-url" in `git remote set-url`
	Action string

	// Flags are optional flags before the positional args
	Flags []Option
	// Args are positional arguments after all flags
	Args []string
	// PostSepArgs are positional args after the "--" separator
	PostSepArgs []string
}

func (sc SubSubCmd) Subcommand() string { return sc.Name }

var actionRegex = regexp.MustCompile(`^[[:alnum:]]+[-[:alnum:]]*$`)

func (sc SubSubCmd) CommandArgs() ([]string, error) {
	var safeArgs []string

	if _, ok := subcommands[sc.Name]; !ok {
		return nil, fmt.Errorf("invalid sub command name %q: %w", sc.Name, ErrInvalidArg)
	}
	safeArgs = append(safeArgs, sc.Name)

	if !actionRegex.MatchString(sc.Action) {
		return nil, fmt.Errorf("invalid sub command action %q: %w", sc.Action, ErrInvalidArg)
	}
	safeArgs = append(safeArgs, sc.Action)

	commandArgs, err := assembleCommandArgs(sc.Name, sc.Flags, sc.Args, sc.PostSepArgs)
	if err != nil {
		return nil, err
	}
	safeArgs = append(safeArgs, commandArgs...)

	return safeArgs, nil
}

// ConfigPair is a sub-command option for use with commands like "git config"
type ConfigPair struct {
	Key   string
	Value string
	// Origin shows the origin type: file, standard input, blob, command line.
	// https://git-scm.com/docs/git-config#Documentation/git-config.txt---show-origin
	Origin string
	// Scope shows the scope of this config value: local, global, system, command.
	// https://git-scm.com/docs/git-config#Documentation/git-config.txt---show-scope
	Scope string
}

var (
	configKeyOptionRegex = regexp.MustCompile(`^[[:alnum:]]+[-[:alnum:]]*\.(.+\.)*[[:alnum:]]+[-[:alnum:]]*$`)
	// configKeyGlobalRegex is intended to verify config keys when used as
	// global arguments. We're playing it safe here by disallowing lots of
	// keys which git would parse just fine, but we only have a limited
	// number of config entries anyway. Most importantly, we cannot allow
	// `=` as part of the key as that would break parsing of `git -c`.
	configKeyGlobalRegex = regexp.MustCompile(`^[[:alnum:]]+(\.[-/_a-zA-Z0-9]+)+$`)
)

// OptionArgs validates the config pair args
func (cp ConfigPair) OptionArgs() ([]string, error) {
	if !configKeyOptionRegex.MatchString(cp.Key) {
		return nil, fmt.Errorf("config key %q failed regexp validation: %w", cp.Key, ErrInvalidArg)
	}
	return []string{cp.Key, cp.Value}, nil
}

// GlobalArgs generates a git `-c <key>=<value>` flag. The key must pass
// validation by containing only alphanumeric sections separated by dots.
// No other characters are allowed for now as `git -c` may not correctly parse
// them, most importantly when they contain equals signs.
func (cp ConfigPair) GlobalArgs() ([]string, error) {
	if !configKeyGlobalRegex.MatchString(cp.Key) {
		return nil, fmt.Errorf("config key %q failed regexp validation: %w", cp.Key, ErrInvalidArg)
	}
	return []string{"-c", fmt.Sprintf("%s=%s", cp.Key, cp.Value)}, nil
}

// Flag is a single token optional command line argument that enables or
// disables functionality (e.g. "-L")
type Flag struct {
	Name string
}

func (f Flag) GlobalArgs() ([]string, error) {
	return f.OptionArgs()
}

// OptionArgs returns an error if the flag is not sanitary
func (f Flag) OptionArgs() ([]string, error) {
	if !flagRegex.MatchString(f.Name) {
		return nil, fmt.Errorf("flag %q failed regex validation: %w", f.Name, ErrInvalidArg)
	}
	return []string{f.Name}, nil
}

// ValueFlag is an optional command line argument that is comprised of pair of
// tokens (e.g. "-n 50")
type ValueFlag struct {
	Name  string
	Value string
}

func (vf ValueFlag) GlobalArgs() ([]string, error) {
	return vf.OptionArgs()
}

// OptionArgs returns an error if the flag is not sanitary
func (vf ValueFlag) OptionArgs() ([]string, error) {
	if !flagRegex.MatchString(vf.Name) {
		return nil, fmt.Errorf("value flag %q failed regex validation: %w", vf.Name, ErrInvalidArg)
	}
	return []string{vf.Name, vf.Value}, nil
}

var flagRegex = regexp.MustCompile(`^(-|--)[[:alnum:]]`)

// IsInvalidArgErr relays if the error is due to an argument validation failure
func IsInvalidArgErr(err error) bool {
	return errors.Is(err, ErrInvalidArg)
}

func validatePositionalArg(arg string) error {
	if strings.HasPrefix(arg, "-") {
		return fmt.Errorf("positional arg %q cannot start with dash '-': %w", arg, ErrInvalidArg)
	}
	return nil
}

// ConvertGlobalOptions converts a protobuf message to command-line flags
func ConvertGlobalOptions(options *gitalypb.GlobalOptions) []GlobalOption {
	var globals []GlobalOption

	if options == nil {
		return globals
	}

	if options.GetLiteralPathspecs() {
		globals = append(globals, Flag{"--literal-pathspecs"})
	}

	return globals
}

type cmdCfg struct {
	env             []string
	globals         []GlobalOption
	stdin           io.Reader
	stdout          io.Writer
	stderr          io.Writer
	hooksConfigured bool
}

// CmdOpt is an option for running a command
type CmdOpt func(*cmdCfg) error

// WithStdin sets the command's stdin.
func WithStdin(r io.Reader) CmdOpt {
	return func(c *cmdCfg) error {
		c.stdin = r
		return nil
	}
}

// WithStdout sets the command's stdout.
func WithStdout(w io.Writer) CmdOpt {
	return func(c *cmdCfg) error {
		c.stdout = w
		return nil
	}
}

// WithStderr sets the command's stderr.
func WithStderr(w io.Writer) CmdOpt {
	return func(c *cmdCfg) error {
		c.stderr = w
		return nil
	}
}

var (
	// ErrRefHookRequired indicates a ref hook configuration is needed but
	// absent from the command
	ErrRefHookRequired = errors.New("ref hook is required but not configured")
	// ErrRefHookNotRequired indicates an extraneous ref hook option was
	// provided
	ErrRefHookNotRequired = errors.New("ref hook is configured but not required")
)

func handleOpts(ctx context.Context, sc Cmd, cc *cmdCfg, opts []CmdOpt) error {
	for _, opt := range opts {
		if err := opt(cc); err != nil {
			return err
		}
	}

	if !cc.hooksConfigured && mayUpdateRef(sc.Subcommand()) {
		return fmt.Errorf("subcommand %q: %w", sc.Subcommand(), ErrRefHookRequired)
	}
	if cc.hooksConfigured && !mayUpdateRef(sc.Subcommand()) {
		return fmt.Errorf("subcommand %q: %w", sc.Subcommand(), ErrRefHookNotRequired)
	}
	if mayGeneratePackfiles(sc.Subcommand()) {
		cc.globals = append(cc.globals, ConfigPair{
			Key: "pack.windowMemory", Value: "100m",
		})
	}

	return nil
}

// SafeCmd creates a command.Command with the given args and Repository. It
// validates the arguments in the command before executing.
func SafeCmd(ctx context.Context, repo repository.GitRepo, globals []GlobalOption, sc Cmd, opts ...CmdOpt) (*command.Command, error) {
	return SafeCmdWithEnv(ctx, nil, repo, globals, sc, opts...)
}

// SafeCmdWithEnv creates a command.Command with the given args, environment, and Repository. It
// validates the arguments in the command before executing.
func SafeCmdWithEnv(ctx context.Context, env []string, repo repository.GitRepo, globals []GlobalOption, sc Cmd, opts ...CmdOpt) (*command.Command, error) {
	cc := &cmdCfg{}

	if err := handleOpts(ctx, sc, cc, opts); err != nil {
		return nil, err
	}

	args, err := combineArgs(globals, sc, cc)
	if err != nil {
		return nil, err
	}

	return NewCommandFactory().unsafeCmdWithEnv(ctx, append(env, cc.env...), cmdStream{
		In:  cc.stdin,
		Out: cc.stdout,
		Err: cc.stderr,
	}, repo, args...)
}

// SafeBareCmd creates a git.Command with the given args and env. It
// validates the arguments in the command before executing.
func SafeBareCmd(ctx context.Context, env []string, globals []GlobalOption, sc Cmd, opts ...CmdOpt) (*command.Command, error) {
	cc := &cmdCfg{}

	if err := handleOpts(ctx, sc, cc, opts); err != nil {
		return nil, err
	}

	args, err := combineArgs(globals, sc, cc)
	if err != nil {
		return nil, err
	}

	return NewCommandFactory().unsafeBareCmd(ctx, cmdStream{
		In:  cc.stdin,
		Out: cc.stdout,
		Err: cc.stderr,
	}, append(env, cc.env...), args...)
}

// SafeBareCmdInDir runs SafeBareCmd in the dir.
func SafeBareCmdInDir(ctx context.Context, dir string, env []string, globals []GlobalOption, sc Cmd, opts ...CmdOpt) (*command.Command, error) {
	if dir == "" {
		return nil, errors.New("no 'dir' provided")
	}

	cc := &cmdCfg{}

	if err := handleOpts(ctx, sc, cc, opts); err != nil {
		return nil, err
	}

	args, err := combineArgs(globals, sc, cc)
	if err != nil {
		return nil, err
	}

	return NewCommandFactory().unsafeBareCmdInDir(ctx, dir, cmdStream{
		In:  cc.stdin,
		Out: cc.stdout,
		Err: cc.stderr,
	}, append(env, cc.env...), args...)
}

// SafeStdinCmd creates a git.Command with the given args and Repository that is
// suitable for Write()ing to. It validates the arguments in the command before
// executing.
func SafeStdinCmd(ctx context.Context, repo repository.GitRepo, globals []GlobalOption, sc Cmd, opts ...CmdOpt) (*command.Command, error) {
	cc := &cmdCfg{}

	if err := handleOpts(ctx, sc, cc, opts); err != nil {
		return nil, err
	}

	args, err := combineArgs(globals, sc, cc)
	if err != nil {
		return nil, err
	}

	return NewCommandFactory().unsafeStdinCmd(ctx, cc.env, repo, args...)
}

// SafeCmdWithoutRepo works like Command but without a git repository. It
// validates the arguments in the command before executing.
func SafeCmdWithoutRepo(ctx context.Context, globals []GlobalOption, sc Cmd, opts ...CmdOpt) (*command.Command, error) {
	cc := &cmdCfg{}

	if err := handleOpts(ctx, sc, cc, opts); err != nil {
		return nil, err
	}

	args, err := combineArgs(globals, sc, cc)
	if err != nil {
		return nil, err
	}

	return NewCommandFactory().unsafeBareCmd(ctx, cmdStream{
		In:  cc.stdin,
		Out: cc.stdout,
		Err: cc.stderr,
	}, cc.env, args...)
}

func combineArgs(globals []GlobalOption, sc Cmd, cc *cmdCfg) (_ []string, err error) {
	var args []string

	defer func() {
		if err != nil && IsInvalidArgErr(err) && len(args) > 0 {
			incrInvalidArg(args[0])
		}
	}()

	for _, global := range append(globals, cc.globals...) {
		globalArgs, err := global.GlobalArgs()
		if err != nil {
			return nil, err
		}
		args = append(args, globalArgs...)
	}

	scArgs, err := sc.CommandArgs()
	if err != nil {
		return nil, err
	}

	return append(args, scArgs...), nil
}
