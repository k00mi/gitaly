package blob

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/rubyserver"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var rubyServer = &rubyserver.Server{}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	defer testhelper.MustHaveNoChildProcess()

	testhelper.Configure()
	testhelper.ConfigureRuby()

	if err := rubyServer.Start(); err != nil {
		log.Fatal(err)
	}
	defer rubyServer.Stop()

	return m.Run()
}

func runBlobServer(t *testing.T, locator storage.Locator) (func(), string) {
	srv := testhelper.NewServer(t, nil, nil)

	gitalypb.RegisterBlobServiceServer(srv.GrpcServer(), NewServer(rubyServer, locator))
	reflection.Register(srv.GrpcServer())

	require.NoError(t, srv.Start())

	return srv.Stop, "unix://" + srv.Socket()
}

func newBlobClient(t *testing.T, serverSocketPath string) (gitalypb.BlobServiceClient, *grpc.ClientConn) {
	connOpts := []grpc.DialOption{
		grpc.WithInsecure(),
	}
	conn, err := grpc.Dial(serverSocketPath, connOpts...)
	if err != nil {
		t.Fatal(err)
	}

	return gitalypb.NewBlobServiceClient(conn), conn
}
