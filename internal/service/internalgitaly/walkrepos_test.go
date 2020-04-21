package internalgitaly

import (
	"io"
	"log"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWalkRepos(t *testing.T) {
	testRoot, clean := testhelper.TempDir(t)
	defer clean()

	log.Printf("testRoot: %s", testRoot)

	storageName := "default"
	storageRoot := filepath.Join(testRoot, "storage")

	server, serverSocketPath := runInternalGitalyServer(t, []config.Storage{
		{Name: storageName, Path: storageRoot},
	})
	defer server.Stop()

	client, conn := newInternalGitalyClient(t, serverSocketPath)
	defer conn.Close()

	testRepo1 := testhelper.NewTestRepoTo(t, storageRoot, "a")
	testRepo2 := testhelper.NewTestRepoTo(t, storageRoot, "b")

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
	require.Contains(t, actualRepos, testRepo1.GetRelativePath())
	require.Contains(t, actualRepos, testRepo2.GetRelativePath())
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
