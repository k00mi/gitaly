package repository

import (
	"os"
	"testing"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"

	"github.com/stretchr/testify/require"
)

func TestSuccessfulFindLicenseRequest(t *testing.T) {
	server, serverSocketPath := runRepoServer(t)
	defer server.Stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	req := &pb.FindLicenseRequest{Repository: testRepo}

	ctx, cancel := testhelper.Context()
	defer cancel()

	resp, err := client.FindLicense(ctx, req)

	require.NoError(t, err)
	require.Equal(t, "mit", resp.GetLicenseShortName())
}

func TestFindLicenseRequestEmptyRepo(t *testing.T) {
	server, serverSocketPath := runRepoServer(t)
	defer server.Stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	ctx, cancel := testhelper.Context()
	defer cancel()

	emptyRepo := &pb.Repository{
		RelativePath: "test-liceense-empty-repo.git",
		StorageName:  testhelper.DefaultStorageName,
	}

	_, err := client.CreateRepository(ctx, &pb.CreateRepositoryRequest{Repository: emptyRepo})
	require.NoError(t, err)

	emptyRepoPath, err := helper.GetRepoPath(emptyRepo)
	require.NoError(t, err)
	defer os.RemoveAll(emptyRepoPath)

	req := &pb.FindLicenseRequest{Repository: emptyRepo}

	resp, err := client.FindLicense(ctx, req)
	require.NoError(t, err)

	require.Empty(t, resp.GetLicenseShortName())
}
