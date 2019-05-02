package catfile_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/git/catfile"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestBatchCacheTTL(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	batchCache := catfile.NewCache(10)

	c, err := catfile.New(ctx, testRepo)
	require.NoError(t, err)

	sessionID := "abcdefg1231231"

	ttl := 5 * time.Millisecond

	cacheKey := catfile.NewCacheKey(sessionID, testRepo)
	batchCache.Add(cacheKey, c, ttl)

	<-time.After(10 * time.Millisecond)

	assert.Nil(t, batchCache.Get(cacheKey))
}

func TestBatchCache(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	batchCache := catfile.NewCache(10)

	c, err := catfile.New(ctx, testRepo)
	require.NoError(t, err)

	sessionID := "abcdefg1231231"

	ttl := 10 * time.Second

	cacheKey := catfile.NewCacheKey(sessionID, testRepo)
	batchCache.Add(cacheKey, c, ttl)
	b := batchCache.Get(cacheKey)
	defer b.Close()

	assert.Equal(t, c, b)
}
