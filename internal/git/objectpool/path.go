package objectpool

import (
	"path/filepath"

	"gitlab.com/gitlab-org/gitaly/internal/git/repository"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

// GetRelativePath will create the relative path to the ObjectPool from the
// storage path.
func (o *ObjectPool) GetRelativePath() string {
	return o.relativePath
}

// GetStorageName exposes the shard name, to satisfy the repository.GitRepo
// interface
func (o *ObjectPool) GetStorageName() string {
	return o.storageName
}

// FullPath on disk, depending on the storage path, and the pools relative path
func (o *ObjectPool) FullPath() string {
	return filepath.Join(o.storagePath, o.GetRelativePath())
}

func alternatesPath(repo repository.GitRepo) (string, error) {
	repoPath, err := helper.GetRepoPath(repo)
	if err != nil {
		return "", err
	}

	return filepath.Join(repoPath, "objects", "info", "alternates"), nil
}
