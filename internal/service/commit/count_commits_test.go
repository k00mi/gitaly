package commit

import (
	"testing"

	"gitlab.com/gitlab-org/gitaly/internal/testhelper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

func TestSuccessfulCountCommitsRequest(t *testing.T) {
	client := newCommitServiceClient(t)

	testCases := []struct {
		revision []byte
		count    int32
	}{
		{
			revision: []byte("1a0b36b3cdad1d2ee32457c102a8c0b7056fa863"),
			count:    1,
		},
		{
			revision: []byte("6d394385cf567f80a8fd85055db1ab4c5295806f"),
			count:    2,
		},
		{
			revision: []byte("e63f41fe459e62e1228fcef60d7189127aeba95a"),
			count:    39,
		},
		{
			revision: []byte("deadfacedeadfacedeadfacedeadfacedeadface"),
			count:    0,
		},
	}

	for _, testCase := range testCases {
		t.Logf("test case: revision=%q count=%d", testCase.revision, testCase.count)

		request := &pb.CountCommitsRequest{
			Repository: testRepo,
			Revision:   testCase.revision,
		}

		response, err := client.CountCommits(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}

		require.Equal(t, response.Count, testCase.count)
	}
}

func TestFailedCountCommitsRequestDueToValidationError(t *testing.T) {
	client := newCommitServiceClient(t)
	revision := []byte("d42783470dc29fde2cf459eb3199ee1d7e3f3a72")

	rpcRequests := []pb.CountCommitsRequest{
		{Repository: &pb.Repository{StorageName: "fake", RelativePath: "path"}, Revision: revision}, // Repository doesn't exist
		{Repository: nil, Revision: revision},                                                       // Repository is nil
		{Repository: testRepo, Revision: nil},                                                       // Revision is empty
	}

	for _, rpcRequest := range rpcRequests {
		t.Logf("test case: %v", rpcRequest)

		_, err := client.CountCommits(context.Background(), &rpcRequest)
		testhelper.AssertGrpcError(t, err, codes.InvalidArgument, "")
	}
}
