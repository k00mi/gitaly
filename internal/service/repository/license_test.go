package repository

import (
	"testing"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
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
	require.Equal(t, "mit", resp.GetLicenseeShortName())
}
