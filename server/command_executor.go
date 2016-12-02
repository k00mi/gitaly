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
	Status string `json:"status"`
	Message string `json:"message"`
}

func CommandExecutorCallback(input []byte) []byte {
	req := cmdRequest{}

	err := json.Unmarshal(input, &req)
	if err != nil {
		return errorResponse("Error parsing JSON request")
	}

	output, err := runCommand(req.Cmd[0], req.Cmd[1:]...)

	if err != nil {
		return errorResponse(string(output.Bytes()))
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

func errorResponse(message string) []byte {
	return makeResponse("error", message)
}

func successResponse(message string) []byte {
	return makeResponse("success", message)
}

func makeResponse(status string, message string) []byte {
	res          := cmdResponse{status, message}
	tempBuf, err := json.Marshal(res)

	if err != nil {
		log.Fatalln("Failed marshalling a JSON response")
	}

	buf := bytes.NewBuffer(tempBuf)
	buf.WriteString("\n")

	return buf.Bytes()
}
