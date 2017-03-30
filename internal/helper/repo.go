package helper

import (
	"fmt"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

// GetRepoPath returns the full path of the repository referenced by an RPC Repository message.
func GetRepoPath(repo *pb.Repository) (string, error) {
	if repo.GetPath() == "" {
		return "", fmt.Errorf("GetRepoPath: empty repo")
	}

	return repo.GetPath(), nil
}
