package objectpool

import (
	"context"

	"gitlab.com/gitlab-org/gitaly/internal/git/repository"
)

// Link writes the alternate file
func Link(ctx context.Context, pool, repository repository.GitRepo) error {
	return nil
}

// Unlink removes the alternate file
func Unlink(ctx context.Context, pool, repository repository.GitRepo) error {
	return nil
}
