package objectpool

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

// NewTestObjectPool creates a new object pool
func NewTestObjectPool(t *testing.T) (*ObjectPool, *gitalypb.Repository) {
	repo, _, relativePath := testhelper.CreateRepo(t, testhelper.GitlabTestStoragePath())

	pool, err := NewObjectPool("default", relativePath)
	require.NoError(t, err)

	return pool, repo
}
