package ref

import (
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

const (
	scratchDir   = "testdata/scratch"
	testRepoRoot = "testdata/data"
	testRepo     = "group/test.git"
)

var serverSocketPath = path.Join(scratchDir, "gitaly.sock")

func TestMain(m *testing.M) {
	source := "https://gitlab.com/gitlab-org/gitlab-test.git"
	clonePath := path.Join(testRepoRoot, testRepo)
	if _, err := os.Stat(clonePath); err != nil {
		testCmd := exec.Command("git", "clone", "--bare", source, clonePath)
		testCmd.Stdout = os.Stdout
		testCmd.Stderr = os.Stderr

		if err := testCmd.Run(); err != nil {
			log.Printf("Test setup: failed to run %v", testCmd)
			os.Exit(-1)
		}
	}

	if err := os.MkdirAll(scratchDir, 0755); err != nil {
		log.Fatal(err)
	}

	os.Exit(func() int {
		return m.Run()
	}())
}

func runRefServer(t *testing.T) *grpc.Server {
	os.Remove(serverSocketPath)
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("unix", serverSocketPath)
	if err != nil {
		t.Fatal(err)
	}

	// Use 100 bytes as the maximum message size to test that fragmenting the ref list works correctly
	pb.RegisterRefServer(grpcServer, &server{MaxMsgSize: 100})
	reflection.Register(grpcServer)

	go grpcServer.Serve(listener)

	return grpcServer
}

func newRefClient(t *testing.T) pb.RefClient {
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

	return pb.NewRefClient(conn)
}
