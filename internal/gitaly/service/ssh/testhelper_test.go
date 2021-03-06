package ssh

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/hook"
	hookservice "gitlab.com/gitlab-org/gitaly/internal/gitaly/service/hook"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc"
)

var (
	gitalySSHPath string
)

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	defer testhelper.MustHaveNoChildProcess()

	defer func(rubyDir string) {
		config.Config.Ruby.Dir = rubyDir
	}(config.Config.Ruby.Dir)

	cleanup := testhelper.Configure()
	defer cleanup()

	testhelper.ConfigureGitalyHooksBinary()
	testhelper.ConfigureGitalySSH()
	gitalySSHPath = filepath.Join(config.Config.BinDir, "gitaly-ssh")

	return m.Run()
}

func runSSHServer(t *testing.T, serverOpts ...ServerOpt) (string, func()) {
	srv := testhelper.NewServer(t, nil, nil, testhelper.WithInternalSocket(config.Config))

	locator := config.NewLocator(config.Config)
	gitalypb.RegisterSSHServiceServer(srv.GrpcServer(), NewServer(locator, serverOpts...))
	gitalypb.RegisterHookServiceServer(srv.GrpcServer(), hookservice.NewServer(config.Config, hook.NewManager(locator, hook.GitlabAPIStub, config.Config)))

	require.NoError(t, srv.Start())

	return "unix://" + srv.Socket(), srv.Stop
}

func newSSHClient(t *testing.T, serverSocketPath string) (gitalypb.SSHServiceClient, *grpc.ClientConn) {
	connOpts := []grpc.DialOption{
		grpc.WithInsecure(),
	}
	conn, err := grpc.Dial(serverSocketPath, connOpts...)
	if err != nil {
		t.Fatal(err)
	}

	return gitalypb.NewSSHServiceClient(conn), conn
}
