package helper

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// Command encapsulates operations with commands creates with NewCommand
type Command struct {
	io.Reader
	*exec.Cmd
}

// Kill cleans the subprocess group of the command. Callers should defer a call
// to kill after they get the command from NewCommand
func (c *Command) Kill() {
	CleanUpProcessGroup(c.Cmd)
}

// GitCommand creates a git Command with the given args
func GitCommand(args ...string) (*Command, error) {
	return NewCommand(exec.Command("git", args...))
}

// NewCommand creates a Command from an exec.Cmd
func NewCommand(cmd *exec.Cmd) (*Command, error) {
	// Explicitly set the environment for the command
	cmd.Env = []string{
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("LD_LIBRARY_PATH=%s", os.Getenv("LD_LIBRARY_PATH")),
	}

	// Start the command in its own process group (nice for signalling)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// If we don't do something with cmd.Stderr, Git errors will be lost
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("GitCommand: stdout: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("GitCommand: start %v: %v", cmd.Args, err)
	}

	return &Command{Reader: stdout, Cmd: cmd}, nil
}
