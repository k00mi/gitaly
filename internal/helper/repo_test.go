package helper

import (
	"testing"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

func TestGetRepoPathWithNilRepo(t *testing.T) {
	if _, err := GetRepoPath(nil); err == nil {
		t.Errorf("Expected an error, got nil")
	}
}

func TestGetRepoPathWithEmptyPath(t *testing.T) {
	repo := &pb.Repository{Path: ""}
	if _, err := GetRepoPath(repo); err == nil {
		t.Errorf("Expected an error, got nil")
	}
}

func TestGetRepoPathWithValidRepo(t *testing.T) {
	expectedPath := "/path/to/repo"
	repo := &pb.Repository{Path: expectedPath}

	path, err := GetRepoPath(repo)
	if err != nil {
		t.Errorf("Expected a nil error, got %v", err)
	}
	if path != expectedPath {
		t.Errorf("Expected path to be %q, got %q", expectedPath, path)
	}
}
