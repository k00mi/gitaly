package testhelper

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const testRepo = "testdata/data/gitlab-test.git"

// MustReadFile returns the content of a file or fails at once.
func MustReadFile(t *testing.T, filename string) []byte {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	return content
}

// GitlabTestRepoPath returns the path to gitlab-test repo.
// Tests should be calling this function instead of cloning the repo themselves.
// Tests that involve modifications to the repo should copy/clone the repo
// via the path returned from this function.
func GitlabTestRepoPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Could not get caller info")
	}

	clonePath := path.Join(path.Dir(currentFile), testRepo)
	if _, err := os.Stat(path.Join(clonePath, "objects")); err != nil {
		log.Fatal("Test repo not found, did you run `make test`?")
	}

	return clonePath
}

// AssertGrpcError asserts the passed err is of the same code as expectedCode. Optionally, it can
// assert the error contains the text of containsText if the latter is not an empty string.
func AssertGrpcError(t *testing.T, err error, expectedCode codes.Code, containsText string) {
	if err == nil {
		t.Fatal("Expected an error, got nil")
	}

	// Check that the code matches
	if code := grpc.Code(err); code != expectedCode {
		t.Fatalf("Expected an error with code %v, got %v. The error was %v", expectedCode, code, err)
	}

	if containsText != "" && !strings.Contains(err.Error(), containsText) {
		t.Fatal(err)
	}
}
