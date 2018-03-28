package repository

import (
	"testing"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"google.golang.org/grpc/codes"

	"github.com/stretchr/testify/require"
)

func TestSuccessfulCalculateChecksum(t *testing.T) {
	server, serverSocketPath := runRepoServer(t)
	defer server.Stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	request := &pb.CalculateChecksumRequest{Repository: testRepo}
	testCtx, cancelCtx := testhelper.Context()
	defer cancelCtx()

	response, err := client.CalculateChecksum(testCtx, request)
	require.NoError(t, err)
	require.Equal(t, "8786527b0747d37d268adc75c5e5e54f3323891c", response.Checksum)
}

func TestEmptyRepositoryCalculateChecksum(t *testing.T) {
	server, serverSocketPath := runRepoServer(t)
	defer server.Stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	repo, _, cleanupFn := testhelper.InitBareRepo(t)
	defer cleanupFn()

	request := &pb.CalculateChecksumRequest{Repository: repo}
	testCtx, cancelCtx := testhelper.Context()
	defer cancelCtx()

	response, err := client.CalculateChecksum(testCtx, request)
	require.NoError(t, err)
	require.Equal(t, "0000000000000000000000000000000000000000", response.Checksum)
}

func TestFailedCalculateChecksum(t *testing.T) {
	server, serverSocketPath := runRepoServer(t)
	defer server.Stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	invalidRepo := &pb.Repository{StorageName: "fake", RelativePath: "path"}

	testCases := []struct {
		desc    string
		request *pb.CalculateChecksumRequest
		code    codes.Code
	}{
		{
			desc:    "Invalid repository",
			request: &pb.CalculateChecksumRequest{Repository: invalidRepo},
			code:    codes.InvalidArgument,
		},
		{
			desc:    "Repository is nil",
			request: &pb.CalculateChecksumRequest{},
			code:    codes.InvalidArgument,
		},
	}

	for _, testCase := range testCases {
		testCtx, cancelCtx := testhelper.Context()
		defer cancelCtx()

		_, err := client.CalculateChecksum(testCtx, testCase.request)
		testhelper.AssertGrpcError(t, err, testCase.code, "")
	}
}
