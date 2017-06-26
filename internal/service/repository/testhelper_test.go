package repository

import (
	"net"
	"os"
	"path"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const scratchDir = "testdata/scratch"

var (
	serverSocketPath = path.Join(scratchDir, "gitaly.sock")
	testRepo         *pb.Repository
)

func TestMain(m *testing.M) {
	testRepo = testhelper.TestRepository()

	if err := os.MkdirAll(scratchDir, 0755); err != nil {
		log.WithError(err).Fatal("mkdirall failed")
	}

	os.Exit(func() int {
		return m.Run()
	}())
}

func newRepositoryClient(t *testing.T) pb.RepositoryServiceClient {
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

	return pb.NewRepositoryServiceClient(conn)
}

func runRepoServer(t *testing.T) *grpc.Server {
	server := grpc.NewServer()
	listener, err := net.Listen("unix", serverSocketPath)
	if err != nil {
		t.Fatal(err)
	}

	pb.RegisterRepositoryServiceServer(server, NewServer())
	reflection.Register(server)

	go server.Serve(listener)

	return server
}
