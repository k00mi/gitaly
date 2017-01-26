package helper

import (
	"log"
	"net/http"
	"os/exec"
	"syscall"
)

func Fail500(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, "Internal server error", 500)
	printError(r, err)
}

func LogError(r *http.Request, err error) {
	printError(r, err)
}

func printError(r *http.Request, err error) {
	if r != nil {
		log.Printf("error: %s %q: %v", r.Method, r.RequestURI, err)
	} else {
		log.Printf("error: %v", err)
	}
}

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
