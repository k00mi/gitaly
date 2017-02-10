package helper

import (
	"os/exec"
	"syscall"
)

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
