package internalgitaly

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serverWrapper struct {
	gitalypb.InternalGitalyServer
	WalkReposFunc func(*gitalypb.WalkReposRequest, gitalypb.InternalGitaly_WalkReposServer) error
}

func (w *serverWrapper) WalkRepos(req *gitalypb.WalkReposRequest, stream gitalypb.InternalGitaly_WalkReposServer) error {
	return w.WalkReposFunc(req, stream)
}

type streamWrapper struct {
	gitalypb.InternalGitaly_WalkReposServer
	SendFunc func(*gitalypb.WalkReposResponse) error
}

func (w *streamWrapper) Send(resp *gitalypb.WalkReposResponse) error {
	return w.SendFunc(resp)
}

func TestWalkRepos(t *testing.T) {
	testRoot, clean := testhelper.TempDir(t)
	defer clean()

	storageName := "default"
	storageRoot := filepath.Join(testRoot, "storage")

	// file walk happens lexicographically, so we delete repository in the middle
	// of the seqeuence to ensure the walk proceeds normally
	testRepo1 := testhelper.NewTestRepoTo(t, storageRoot, "a")
	deletedRepo := testhelper.NewTestRepoTo(t, storageRoot, "b")
	testRepo2 := testhelper.NewTestRepoTo(t, storageRoot, "c")

	// to test a directory being deleted during a walk, we must delete a directory after
	// the file walk has started. To achieve that, we wrap the server to pass down a wrapped
	// stream that allows us to hook in to stream responses. We then delete 'b' when
	// the first repo 'a' is being streamed to the client.
	deleteOnce := sync.Once{}
	srv := NewServer([]config.Storage{{Name: storageName, Path: storageRoot}})
	wsrv := &serverWrapper{srv,
		func(r *gitalypb.WalkReposRequest, s gitalypb.InternalGitaly_WalkReposServer) error {
			return srv.WalkRepos(r, &streamWrapper{s,
				func(resp *gitalypb.WalkReposResponse) error {
					deleteOnce.Do(func() {
						require.NoError(t, os.RemoveAll(filepath.Join(storageRoot, deletedRepo.RelativePath)))
					})
					return s.Send(resp)
				},
			})
		},
	}

	server, serverSocketPath := runInternalGitalyServer(t, wsrv)
	defer server.Stop()

	client, conn := newInternalGitalyClient(t, serverSocketPath)
	defer conn.Close()

	ctx, cancel := testhelper.Context()
	defer cancel()

	stream, err := client.WalkRepos(ctx, &gitalypb.WalkReposRequest{
		StorageName: "invalid storage name",
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.NotNil(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, s.Code())

	stream, err = client.WalkRepos(ctx, &gitalypb.WalkReposRequest{
		StorageName: storageName,
	})
	require.NoError(t, err)

	actualRepos := consumeWalkReposStream(t, stream)
	require.Equal(t, []string{
		testRepo1.GetRelativePath(),
		testRepo2.GetRelativePath(),
	}, actualRepos)
}

func consumeWalkReposStream(t *testing.T, stream gitalypb.InternalGitaly_WalkReposClient) []string {
	var repos []string
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		} else {
			require.NoError(t, err)
		}
		repos = append(repos, resp.RelativePath)
	}
	return repos
}
