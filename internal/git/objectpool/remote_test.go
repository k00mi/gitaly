package objectpool

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestRemoveRemote(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	pool, err := NewObjectPool(testRepo.GetStorageName(), t.Name())
	require.NoError(t, err)

	require.NoError(t, pool.clone(ctx, testRepo))
	defer pool.Remove(ctx)

	require.NoError(t, pool.removeRemote(ctx, "origin"))

	out := testhelper.MustRunCommand(t, nil, "git", "-C", pool.FullPath(), "remote")
	require.Len(t, out, 0)
}
