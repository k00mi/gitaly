package repository

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

var (
	freshTime   = time.Now()
	oldTime     = freshTime.Add(-2 * time.Hour)
	oldTreeTime = freshTime.Add(-7 * time.Hour)
)

func TestGarbageCollectSuccess(t *testing.T) {
	server, serverSocketPath := runRepoServer(t)
	defer server.Stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	tests := []struct {
		req  *pb.GarbageCollectRequest
		desc string
	}{
		{
			req:  &pb.GarbageCollectRequest{Repository: testRepo, CreateBitmap: false},
			desc: "without bitmap",
		},
		{
			req:  &pb.GarbageCollectRequest{Repository: testRepo, CreateBitmap: true},
			desc: "with bitmap",
		},
	}

	packPath := path.Join(testhelper.GitlabTestStoragePath(), testRepo.GetRelativePath(), "objects", "pack")

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Reset mtime to a long while ago since some filesystems don't have sub-second
			// precision on `mtime`.
			// Stamp taken from https://golang.org/pkg/time/#pkg-constants
			testhelper.MustRunCommand(t, nil, "touch", "-t", testTimeString, packPath)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c, err := client.GarbageCollect(ctx, test.req)
			assert.NoError(t, err)
			assert.NotNil(t, c)

			// Entire `path`-folder gets updated so this is fine :D
			assertModTimeAfter(t, testTime, packPath)

			bmPath, err := filepath.Glob(path.Join(packPath, "pack-*.bitmap"))
			if err != nil {
				t.Fatalf("Error globbing bitmaps: %v", err)
			}
			if test.req.GetCreateBitmap() {
				if len(bmPath) == 0 {
					t.Errorf("No bitmaps found")
				}
			} else {
				if len(bmPath) != 0 {
					t.Errorf("Bitmap found: %v", bmPath)
				}
			}
		})
	}
}

func TestGarbageCollectDeletesRefsLocks(t *testing.T) {
	server, serverSocketPath := runRepoServer(t)
	defer server.Stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	ctx, cancel := testhelper.Context()
	defer cancel()

	req := &pb.GarbageCollectRequest{Repository: testRepo}
	refsPath := filepath.Join(testRepoPath, "refs")

	// Note: Creating refs this way makes `git gc` crash but this actually works
	// in our favor for this test since we can ensure that the files kept and
	// deleted are all due to our *.lock cleanup step before gc runs (since
	// `git gc` also deletes files from /refs when packing).
	keepRefPath := filepath.Join(refsPath, "heads", "keepthis")
	createFileWithTimes(keepRefPath, freshTime)
	keepOldRefPath := filepath.Join(refsPath, "heads", "keepthisalso")
	createFileWithTimes(keepOldRefPath, oldTime)
	keepDeceitfulRef := filepath.Join(refsPath, "heads", " .lock.not-actually-a-lock.lock ")
	createFileWithTimes(keepDeceitfulRef, oldTime)

	keepLockPath := filepath.Join(refsPath, "heads", "keepthis.lock")
	createFileWithTimes(keepLockPath, freshTime)

	deleteLockPath := filepath.Join(refsPath, "heads", "deletethis.lock")
	createFileWithTimes(deleteLockPath, oldTime)

	c, err := client.GarbageCollect(ctx, req)
	testhelper.AssertGrpcError(t, err, codes.Internal, "GarbageCollect: cmd wait")
	assert.Nil(t, c)

	// Sanity checks
	assert.FileExists(t, keepRefPath)
	assert.FileExists(t, keepOldRefPath)
	assert.FileExists(t, keepDeceitfulRef)

	assert.FileExists(t, keepLockPath)

	// There's assert.FileExists but no assert.NotFileExists ¯\_(ツ)_/¯
	_, err = os.Stat(deleteLockPath)
	assert.True(t, os.IsNotExist(err))
}

func TestGarbageCollectDeletesPackedRefsLock(t *testing.T) {
	server, serverSocketPath := runRepoServer(t)
	defer server.Stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	testCases := []struct {
		desc        string
		lockTime    *time.Time
		shouldExist bool
	}{
		{
			desc:        "with a recent lock",
			lockTime:    &freshTime,
			shouldExist: true,
		},
		{
			desc:        "with an old lock",
			lockTime:    &oldTime,
			shouldExist: false,
		},
		{
			desc:        "with a non-existing lock",
			lockTime:    nil,
			shouldExist: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
			defer cleanupFn()

			// Force the packed-refs file to have an old time to test that even
			// in that case it doesn't get deleted
			packedRefsPath := filepath.Join(testRepoPath, "packed-refs")
			os.Chtimes(packedRefsPath, oldTime, oldTime)

			req := &pb.GarbageCollectRequest{Repository: testRepo}
			lockPath := filepath.Join(testRepoPath, "packed-refs.lock")

			if tc.lockTime != nil {
				createFileWithTimes(lockPath, *tc.lockTime)
			}

			ctx, cancel := testhelper.Context()
			defer cancel()

			c, err := client.GarbageCollect(ctx, req)

			// Sanity checks
			assert.FileExists(t, filepath.Join(testRepoPath, "HEAD")) // For good measure
			assert.FileExists(t, packedRefsPath)

			if tc.shouldExist {
				assert.FileExists(t, lockPath)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, c)

				testhelper.AssertFileNotExists(t, lockPath)
			}
		})
	}
}

func TestGarbageCollectDeletesStaleWorktrees(t *testing.T) {
	server, serverSocketPath := runRepoServer(t)
	defer server.Stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	testCases := []struct {
		desc         string
		worktreeTime time.Time
		shouldExist  bool
	}{
		{
			desc:         "with a recent worktree",
			worktreeTime: freshTime,
			shouldExist:  true,
		},
		{
			desc:         "with a slightly old worktree",
			worktreeTime: oldTime,
			shouldExist:  true,
		},
		{
			desc:         "with an old worktree",
			worktreeTime: oldTreeTime,
			shouldExist:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
			defer cleanupFn()

			req := &pb.GarbageCollectRequest{Repository: testRepo}

			testhelper.AddWorktree(t, testRepoPath, "test-worktree")
			basePath := filepath.Join(testRepoPath, "worktrees")
			worktreePath := filepath.Join(basePath, "test-worktree")

			require.NoError(t, os.Chtimes(worktreePath, tc.worktreeTime, tc.worktreeTime))

			ctx, cancel := testhelper.Context()
			defer cancel()

			c, err := client.GarbageCollect(ctx, req)

			// Sanity check
			assert.FileExists(t, filepath.Join(testRepoPath, "HEAD")) // For good measure

			if tc.shouldExist {
				assert.DirExists(t, basePath)
				assert.DirExists(t, worktreePath)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, c)

				testhelper.AssertFileNotExists(t, worktreePath)
			}
		})
	}
}

func TestGarbageCollectFailure(t *testing.T) {
	server, serverSocketPath := runRepoServer(t)
	defer server.Stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	tests := []struct {
		repo *pb.Repository
		code codes.Code
	}{
		{repo: nil, code: codes.InvalidArgument},
		{repo: &pb.Repository{StorageName: "foo"}, code: codes.InvalidArgument},
		{repo: &pb.Repository{RelativePath: "bar"}, code: codes.InvalidArgument},
		{repo: &pb.Repository{StorageName: testRepo.GetStorageName(), RelativePath: "bar"}, code: codes.NotFound},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test.repo), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, err := client.GarbageCollect(ctx, &pb.GarbageCollectRequest{Repository: test.repo})
			testhelper.AssertGrpcError(t, err, test.code, "")
		})
	}

}

func createFileWithTimes(path string, mTime time.Time) {
	ioutil.WriteFile(path, nil, 0644)
	os.Chtimes(path, mTime, mTime)
}
