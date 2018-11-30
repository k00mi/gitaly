package objectpool

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"gitlab.com/gitlab-org/gitaly/internal/git/repository"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

// Link will write the relative path to the object pool from the repository that
// is to join the pool. This does not trigger deduplication, which is the
// responsibility of the caller.
func (o *ObjectPool) Link(ctx context.Context, repo repository.GitRepo) error {
	altPath, err := alternatesPath(repo)
	if err != nil {
		return err
	}

	repoPath, err := helper.GetRepoPath(repo)
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(filepath.Join(repoPath, "objects"), o.FullPath())
	if err != nil {
		return err
	}

	return ioutil.WriteFile(altPath, []byte(filepath.Join(relPath, "objects")), 0644)
}

// Unlink removes the alternates file, so Git won't look there anymore
func Unlink(ctx context.Context, repo repository.GitRepo) error {
	altPath, err := alternatesPath(repo)
	if err != nil {
		return err
	}

	return os.RemoveAll(altPath)
}
