package notifications

import (
	"log"
	"net"
	"os"
	"path"
	"testing"
	"time"

	pb "gitlab.com/gitlab-org/gitaly/protos/go"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	scratchDir   = "testdata/scratch"
	testRepoRoot = "testdata/data"
	testRepo     = "group/test.git"
)

var serverSocketPath = path.Join(scratchDir, "gitaly.sock")

func TestMain(m *testing.M) {
	if err := os.MkdirAll(scratchDir, 0755); err != nil {
		log.Fatal(err)
	}

	os.Exit(func() int {
		return m.Run()
	}())
}

func TestSuccessfulPostReceive(t *testing.T) {
	server := runNotificationsServer(t)
	defer server.Stop()

	client := newNotificationsClient(t)
	repo := &pb.Repository{Path: path.Join(testRepoRoot, testRepo)}
	rpcRequest := &pb.PostReceiveRequest{Repository: repo}

	_, err := client.PostReceive(context.Background(), rpcRequest)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEmptyPostReceiveRequest(t *testing.T) {
	server := runNotificationsServer(t)
	defer server.Stop()

	client := newNotificationsClient(t)
	rpcRequest := &pb.PostReceiveRequest{}

	_, err := client.PostReceive(context.Background(), rpcRequest)
	if err.Error() != "rpc error: code = 2 desc = Bad Request (empty repository)" {
		t.Fatal(err)
	}
}

func runNotificationsServer(t *testing.T) *grpc.Server {
	server := grpc.NewServer()
	listener, err := net.Listen("unix", serverSocketPath)
	if err != nil {
		t.Fatal(err)
	}

	pb.RegisterNotificationsServer(server, NewServer())
	reflection.Register(server)

	go server.Serve(listener)

	return server
}

func newNotificationsClient(t *testing.T) pb.NotificationsClient {
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

	return pb.NewNotificationsClient(conn)
}
