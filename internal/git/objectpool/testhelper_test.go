package objectpool

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func NewTestObjectPool(ctx context.Context, t *testing.T, storageName string) (*ObjectPool, func()) {
	pool, err := NewObjectPool(storageName, testhelper.NewTestObjectPoolName(t))
	require.NoError(t, err)
	return pool, func() {
		require.NoError(t, pool.Remove(ctx))
	}
}
