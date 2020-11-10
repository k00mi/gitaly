package ref

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/helper/lines"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	localBranches = map[string]*gitalypb.GitCommit{
		"refs/heads/100%branch":      testhelper.GitLabTestCommit("1b12f15a11fc6e62177bef08f47bc7b5ce50b141"),
		"refs/heads/improve/awesome": testhelper.GitLabTestCommit("5937ac0a7beb003549fc5fd26fc247adbce4a52e"),
		"refs/heads/'test'":          testhelper.GitLabTestCommit("e56497bb5f03a90a51293fc6d516788730953899"),
	}
)

func TestMain(m *testing.M) {
	testhelper.Configure()
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	defer testhelper.MustHaveNoChildProcess()

	// Force small messages to test that fragmenting the
	// ref list works correctly
	lines.ItemsPerMessage = 3

	return m.Run()
}

func runRefServiceServer(t *testing.T) (func(), string) {
	srv := testhelper.NewServer(t, nil, nil)

	gitalypb.RegisterRefServiceServer(srv.GrpcServer(), NewServer(config.NewLocator(config.Config)))
	reflection.Register(srv.GrpcServer())

	require.NoError(t, srv.Start())

	return srv.Stop, "unix://" + srv.Socket()
}

func newRefServiceClient(t *testing.T, serverSocketPath string) (gitalypb.RefServiceClient, *grpc.ClientConn) {
	connOpts := []grpc.DialOption{
		grpc.WithInsecure(),
	}
	conn, err := grpc.Dial(serverSocketPath, connOpts...)
	if err != nil {
		t.Fatal(err)
	}

	return gitalypb.NewRefServiceClient(conn), conn
}

func assertContainsLocalBranch(t *testing.T, branches []*gitalypb.FindLocalBranchResponse, branch *gitalypb.FindLocalBranchResponse) {
	t.Helper()

	for _, b := range branches {
		if bytes.Equal(branch.Name, b.Name) {
			if !testhelper.FindLocalBranchResponsesEqual(branch, b) {
				t.Errorf("Expected branch\n%v\ngot\n%v", branch, b)
			}

			testhelper.ProtoEqual(t, branch.Commit, b.Commit)
			return // Found the branch and it maches. Success!
		}
	}
	t.Errorf("Expected to find branch %q in local branches", branch.Name)
}

func assertContainsBranch(t *testing.T, branches []*gitalypb.FindAllBranchesResponse_Branch, branch *gitalypb.FindAllBranchesResponse_Branch) {
	t.Helper()

	var branchNames [][]byte

	for _, b := range branches {
		if bytes.Equal(branch.Name, b.Name) {
			testhelper.ProtoEqual(t, b.Target, branch.Target)
			return // Found the branch and it maches. Success!
		}
		branchNames = append(branchNames, b.Name)
	}

	t.Errorf("Expected to find branch %q in branches %s", branch.Name, branchNames)
}
