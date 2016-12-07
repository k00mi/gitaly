package server

import (
	"bufio"
	"bytes"
	"net"
	"os"
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
	res := responseForCommand(`{"cmd":["ls", "-hal"]}`, t)

	if !bytes.Contains(res, []byte(`{"status":"success","message":"total`)) {
		t.Fatalf("Expected a successful response, got this response: %s", res)
	}

	if !bytes.Contains(res, []byte(`"exit_status":0`)) {
		t.Fatalf("Expected response to contain exit status of 0, found none in: %s", res)
	}
}

func TestRunningCommandUnsuccessfully(t *testing.T) {
	res := responseForCommand(`{"cmd":["ls", "/file-that-does-not-exist"]}`, t)

	if !bytes.Contains(res, []byte(`{"status":"error","message":"ls: cannot access`)) {
		t.Fatalf("Expected a failure response, got this response: %s", res)
	}

	if !bytes.Contains(res, []byte(`"exit_status":2`)) {
		t.Fatalf("Expected response to contain exit status of 2, found none in: %s", res)
	}
}

func TestMalformedCommand(t *testing.T) {
	res := responseForCommand(`{"cmd":["ls", "/file-that-does-not-exist"}`, t)

	if !bytes.Equal(res, []byte(`{"status":"error","message":"Error parsing JSON request","exit_status":255}`)) {
		t.Fatalf("Expected a failure response, got this response: %s", res)
	}
}

func responseForCommand(cmd string, t *testing.T) []byte {
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
