package ref

import (
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/stretchr/testify/require"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestSuccessfulDeleteRefs(t *testing.T) {
	server, serverSocketPath := runRefServiceServer(t)
	defer server.Stop()

	client, conn := newRefServiceClient(t, serverSocketPath)
	defer conn.Close()

	repo, repoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "update-ref", "refs/delete/a", "b83d6e391c22777fca1ed3012fce84f633d7fed0")
	testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "update-ref", "refs/also-delete/b", "1b12f15a11fc6e62177bef08f47bc7b5ce50b141")
	testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "update-ref", "refs/keep/c", "498214de67004b1da3d820901307bed2a68a8ef6")
	testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "update-ref", "refs/also-keep/d", "b83d6e391c22777fca1ed3012fce84f633d7fed0")

	rpcRequest := &pb.DeleteRefsRequest{
		Repository:       repo,
		ExceptWithPrefix: [][]byte{[]byte("refs/keep"), []byte("refs/also-keep"), []byte("refs/heads/")},
	}

	ctx, cancel := testhelper.Context()
	defer cancel()

	_, err := client.DeleteRefs(ctx, rpcRequest)
	require.NoError(t, err)

	refs := testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "for-each-ref")
	refsStr := string(refs)

	require.NotContains(t, refsStr, "refs/delete/a")
	require.NotContains(t, refsStr, "refs/also-delete/b")
	require.Contains(t, refsStr, "refs/keep/c")
	require.Contains(t, refsStr, "refs/also-keep/d")
	require.Contains(t, refsStr, "refs/heads/master")
}

func TestFailedDeleteRefsDueToValidation(t *testing.T) {
	server, serverSocketPath := runRefServiceServer(t)
	defer server.Stop()

	client, conn := newRefServiceClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	testCases := []struct {
		desc     string
		repo     *pb.Repository
		prefixes [][]byte
		code     codes.Code
	}{
		{
			desc:     "Invalid repository",
			repo:     &pb.Repository{StorageName: "fake", RelativePath: "path"},
			prefixes: [][]byte{[]byte("exclude-this")},
			code:     codes.InvalidArgument,
		},
		{
			desc:     "Repository is nil",
			repo:     nil,
			prefixes: [][]byte{[]byte("exclude-this")},
			code:     codes.InvalidArgument,
		},
		{
			desc:     "No prefixes",
			repo:     testRepo,
			prefixes: [][]byte{},
			code:     codes.InvalidArgument,
		},
		{
			desc:     "Empty prefix",
			repo:     testRepo,
			prefixes: [][]byte{[]byte("exclude-this"), []byte{}},
			code:     codes.InvalidArgument,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			rpcRequest := &pb.DeleteRefsRequest{
				Repository:       tc.repo,
				ExceptWithPrefix: tc.prefixes,
			}
			_, err := client.DeleteRefs(ctx, rpcRequest)
			testhelper.AssertGrpcError(t, err, tc.code, "")
		})
	}
}
