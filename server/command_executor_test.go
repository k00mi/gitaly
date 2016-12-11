package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

const serviceAddress = "127.0.0.1:6667"

func TestMain(m *testing.M) {
	service := NewService()

	go service.Serve(serviceAddress, CommandExecutorCallback)
	defer service.Stop()

	time.Sleep(10 * time.Millisecond)
	os.Exit(m.Run())
}

func TestRunningCommandSuccessfully(t *testing.T) {
	res := responseFromCommand(`{"cmd":["ls", "-hal"]}`, t)

	if res.ExitStatus != 0 {
		t.Error("Expected exit status of 0, found %v", res.ExitStatus)
	}

	if res.Status != "success" { // We should have these statuses as constants in the package
		t.Error("Expected status of success, found %v", res.Status)
	}
}

func TestRunningCommandUnsuccessfully(t *testing.T) {
	res := responseFromCommand(`{"cmd":["ls", "/file-that-does-not-exist"]}`, t)

	if res.ExitStatus == 0 {
		t.Error("Expected a failure exit status, got 0")
	}

	if res.Status != "error" {
		t.Error("Expected error status, got %v", res.Status)
	}

	if !(strings.Contains(res.Message, "cannot access") ||
		strings.Contains(res.Message, "No such file or directory")) {
		t.Error("Expected ls error message, got %v", res.Message)
	}
}

func TestMalformedCommand(t *testing.T) {
	res := responseFromCommand(`{"cmd":["ls", "/file-that-does-not-exist"}`, t)

	if res.Status != "error" {
		t.Error("Expected error status, got %s", res.Status)
	}

	if res.Message != "Error parsing JSON request" {
		t.Error("Expected parsing json error message, got %s", res.Message)
	}

	if res.ExitStatus != 255 {
		t.Error("Expected exit status 255, got %v", res.ExitStatus)
	}
}

// These 2 functions could be interesting to reuse in the client
// For this to happen we should remove the testing dependency and
// then move them to an accessible place.
func responseFromCommand(cmd string, t *testing.T) CmdResponse {
	var response CmdResponse
	buffer := bytesFromCommand(cmd, t)
	err := json.Unmarshal(buffer, &response)
	if err != nil {
		t.Error(err)
	}
	return response
}

func bytesFromCommand(cmd string, t *testing.T) []byte {
	conn, err := net.Dial("tcp", serviceAddress)
	if err != nil {
		t.Fatal(err)
	}

	defer conn.Close()

	if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
		t.Error(err)
	}

	reader := bufio.NewReader(conn)
	buffer, err := reader.ReadBytes('\n')
	if err != nil {
		t.Error(err)
	}

	return bytes.TrimSpace(buffer)
}
