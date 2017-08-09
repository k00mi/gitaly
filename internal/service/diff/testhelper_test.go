package diff

import (
	"net"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

var testRepo *pb.Repository

func TestMain(m *testing.M) {
	testRepo = testhelper.TestRepository()

	os.Exit(func() int {
		return m.Run()
	}())
}

func newDiffClient(t *testing.T) (pb.DiffServiceClient, *grpc.ClientConn) {
	connOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, _ time.Duration) (net.Conn, error) {
			return net.Dial("unix", addr)
		}),
	}
	conn, err := grpc.Dial(serverSocketPath, connOpts...)
	if err != nil {
		t.Fatal(err)
	}

	return pb.NewDiffServiceClient(conn), conn
}
