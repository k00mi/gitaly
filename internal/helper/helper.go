package helper

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// GitCommand returns a Cmd that's ready to be run by Cmd.Run. It sets
// the proper environment variables along with other tweaks.
func GitCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	// Start the command in its own process group (nice for signalling)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Explicitly set the environment for the Git command
	cmd.Env = []string{
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("LD_LIBRARY_PATH=%s", os.Getenv("LD_LIBRARY_PATH")),
	}
	// If we don't do something with cmd.Stderr, Git errors will be lost
	cmd.Stderr = os.Stderr
	return cmd
}

// CleanUpProcessGroup will send a SIGTERM signal to the process group
// belonging to the `cmd` process
func CleanUpProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}

	process := cmd.Process
	if process != nil && process.Pid > 0 {
		// Send SIGTERM to the process group of cmd
		syscall.Kill(-process.Pid, syscall.SIGTERM)
	}

	// reap our child process
	cmd.Wait()
}
