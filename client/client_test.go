package client

import (
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"gitlab.com/gitlab-org/git-access-daemon/server"
)

const serviceAddress = "127.0.0.1:6667"
const testRepo = "group/test.git"
const testRepoRoot = "testdata/data"

func TestMain(m *testing.M) {
	source := "https://gitlab.com/gitlab-org/gitlab-test.git"
	clonePath := path.Join(testRepoRoot, testRepo)
	if _, err := os.Stat(clonePath); err != nil {
		testCmd := exec.Command("git", "clone", "--bare", source, clonePath)
		testCmd.Stdout = os.Stdout
		testCmd.Stderr = os.Stderr
		if err := testCmd.Run(); err != nil {
			log.Printf("Test setup: failed to run %v", testCmd)
			os.Exit(-1)
		}
	}
	service := server.NewService()

	go service.Serve(serviceAddress, server.CommandExecutorCallback)
	defer service.Stop()

	time.Sleep(10 * time.Millisecond)
	os.Exit(m.Run())
}

func TestRunningGitCommandSuccessfully(t *testing.T) {
	client := NewClient(serviceAddress)
	defer client.Close()

	res := client.Request([]string{
		"git",
		"--git-dir",
		path.Join(testRepoRoot, testRepo),
		"rev-list",
		"--count",
		"b83d6e391c",
	})

	exit_status := 0
	if res.ExitStatus != exit_status {
		t.Fatalf("Expected response exit status to equal %d, got %d", exit_status, res.ExitStatus)
	}

	msg := "37\n"
	if res.Message != msg {
		t.Fatalf("Expected response stdout to be \"%s\", got \"%s\"", msg, res.Message)
	}
}

func TestRunningGitCommandUnsuccessfully(t *testing.T) {
	client := NewClient(serviceAddress)
	defer client.Close()

	res := client.Request([]string{
		"git",
		"--git-dir",
		path.Join(testRepoRoot, testRepo),
		"rev-list",
		"--count",
		"babecafe",
	})

	exit_status := 128
	if res.ExitStatus != exit_status {
		t.Fatalf("Expected response exit status to equal %d, got %d", exit_status, res.ExitStatus)
	}

	msg := "fatal: ambiguous argument 'babecafe': unknown revision or path not in the working tree."
	if !strings.Contains(res.Message, msg) {
		t.Fatalf("Expected stderr to contain \"%s\", found none in \"%s\"", msg, res.Message)
	}
}
