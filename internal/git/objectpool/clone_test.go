package objectpool

import (
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestClone(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	pool, err := NewObjectPool(testRepo.GetStorageName(), "@pools"+t.Name())
	require.NoError(t, err)

	err = pool.clone(ctx, testRepo)
	require.NoError(t, err)
	defer pool.Remove(ctx)

	require.DirExists(t, pool.FullPath())
	require.DirExists(t, path.Join(pool.FullPath(), "objects"))
}

func TestRemoveRefs(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	pool, err := NewObjectPool(testRepo.GetStorageName(), t.Name())
	require.NoError(t, err)
	defer pool.Remove(ctx)

	require.NoError(t, pool.clone(ctx, testRepo))
	require.NoError(t, pool.removeRefs(ctx))

	out := testhelper.MustRunCommand(t, nil, "git", "-C", pool.FullPath(), "for-each-ref")
	require.Len(t, out, 0)
}

func TestCloneExistingPool(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	pool, err := NewObjectPool(testRepo.GetStorageName(), t.Name())
	require.NoError(t, err)

	err = pool.clone(ctx, testRepo)
	require.NoError(t, err)
	defer pool.Remove(ctx)

	// Reclone on the directory
	err = pool.clone(ctx, testRepo)
	require.Error(t, err)
}
