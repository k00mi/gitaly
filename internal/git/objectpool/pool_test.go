package objectpool

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestNewObjectPool(t *testing.T) {
	_, err := NewObjectPool("default", t.Name())
	require.NoError(t, err)

	_, err = NewObjectPool("mepmep", t.Name())
	require.Error(t, err)
}

func TestCreate(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	pool, err := NewObjectPool(testRepo.GetStorageName(), t.Name())
	require.NoError(t, err)

	err = pool.Create(ctx, testRepo)
	require.NoError(t, err)
	defer pool.Remove(ctx)

	require.True(t, pool.IsValid())

	// No refs
	out := testhelper.MustRunCommand(t, nil, "git", "-C", pool.FullPath(), "for-each-ref")
	assert.Len(t, out, 0)

	// No remotes
	out = testhelper.MustRunCommand(t, nil, "git", "-C", pool.FullPath(), "remote")
	assert.Len(t, out, 0)

	// No hooks
	_, err = os.Stat(path.Join(pool.FullPath(), "hooks"))
	assert.True(t, os.IsNotExist(err))

	// No problems
	out = testhelper.MustRunCommand(t, nil, "git", "-C", pool.FullPath(), "cat-file", "-s", "55bc176024cfa3baaceb71db584c7e5df900ea65")
	assert.Equal(t, "282\n", string(out))
}

func TestCreateSubDirsExist(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	pool, err := NewObjectPool(testRepo.GetStorageName(), t.Name())
	defer pool.Remove(ctx)
	require.NoError(t, err)

	err = pool.Create(ctx, testRepo)
	require.NoError(t, err)

	pool.Remove(ctx)

	// Recreate pool so the subdirs exist already
	err = pool.Create(ctx, testRepo)
	require.NoError(t, err)
}

func TestRemove(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	pool, err := NewObjectPool(testRepo.GetStorageName(), t.Name())
	require.NoError(t, err)

	err = pool.Create(ctx, testRepo)
	require.NoError(t, err)

	require.True(t, pool.Exists())
	require.NoError(t, pool.Remove(ctx))
	require.False(t, pool.Exists())
}
