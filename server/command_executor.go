package server

import (
	"bytes"
	"encoding/json"
	"log"
	"os/exec"
	"syscall"
)

type cmdRequest struct {
	Cmd []string `json:"cmd"`
}

type cmdResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	ExitStatus int    `json:"exit_status"`
}

func CommandExecutorCallback(input []byte) []byte {
	req := cmdRequest{}

	err := json.Unmarshal(input, &req)
	if err != nil {
		return errorResponse("Error parsing JSON request", 255)
	}

	output, err := runCommand(req.Cmd[0], req.Cmd[1:]...)

	if err != nil {
		return errorResponse(
			string(output.Bytes()),
			extractExitStatusFromError(err.(*exec.ExitError)),
		)
	}

	return successResponse(string(output.Bytes()))
}

func runCommand(name string, args ...string) (bytes.Buffer, error) {
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	log.Println("Executing command:", name, "with args", args)

	cmd := makeCommand(name, args...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		return stderrBuf, err
	}

	return stdoutBuf, nil
}

// Based on git.gitCommand from gitlab-workhorse
func makeCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)

	// Start the command in its own process group (nice for signalling)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd
}

func extractExitStatusFromError(err *exec.ExitError) int {
	processState := err.ProcessState
	status := processState.Sys().(syscall.WaitStatus)

	if status.Exited() {
		return status.ExitStatus()
	}

	return 255
}

func errorResponse(message string, exit_status int) []byte {
	return makeResponse("error", message, exit_status)
}

func successResponse(message string) []byte {
	return makeResponse("success", message, 0)
}

func makeResponse(status string, message string, exit_status int) []byte {
	res := cmdResponse{status, message, exit_status}
	tempBuf, err := json.Marshal(res)

	if err != nil {
		log.Fatalln("Failed marshalling a JSON response")
	}

	buf := bytes.NewBuffer(tempBuf)
	buf.WriteString("\n")

	return buf.Bytes()
}
