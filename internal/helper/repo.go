package helper

import (
	"fmt"
	"path"

	"gitlab.com/gitlab-org/gitaly/internal/config"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

// GetRepoPath returns the full path of the repository referenced by an RPC Repository message.
func GetRepoPath(repo *pb.Repository) (string, error) {
	if storagePath, ok := config.StoragePath(repo.GetStorageName()); ok {
		return path.Join(storagePath, repo.GetRelativePath()), nil
	}

	if repo.GetPath() == "" {
		return "", fmt.Errorf("GetRepoPath: empty repo")
	}

	return repo.GetPath(), nil
}
