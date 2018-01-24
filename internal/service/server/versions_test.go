package server

import (
	"net"
	"testing"
	"time"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	gitalyauth "gitlab.com/gitlab-org/gitaly/auth"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/server/auth"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/internal/version"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func TestGitalyServerVersion(t *testing.T) {
	server, serverSocketPath := runServer(t)
	defer server.Stop()

	client, conn := newServerClient(t, serverSocketPath)
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := client.ServerVersion(ctx, &pb.ServerVersionRequest{})
	require.NoError(t, err)

	require.Equal(t, version.GetVersion(), c.GetVersion())
}

func TestGitalyServerGitVersion(t *testing.T) {
	server, serverSocketPath := runServer(t)
	defer server.Stop()

	client, conn := newServerClient(t, serverSocketPath)
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := client.ServerGitVersion(ctx, &pb.ServerGitVersionRequest{})
	require.NoError(t, err)

	version, err := git.Version()
	require.NoError(t, err)
	require.Equal(t, version, c.GetVersion())
}

func runServer(t *testing.T) (*grpc.Server, string) {
	streamInt := []grpc.StreamServerInterceptor{auth.StreamServerInterceptor()}
	unaryInt := []grpc.UnaryServerInterceptor{auth.UnaryServerInterceptor()}

	server := testhelper.NewTestGrpcServer(t, streamInt, unaryInt)
	serverSocketPath := testhelper.GetTemporaryGitalySocketFileName()

	listener, err := net.Listen("unix", serverSocketPath)
	if err != nil {
		t.Fatal(err)
	}

	pb.RegisterServerServiceServer(server, NewServer())
	reflection.Register(server)

	go server.Serve(listener)

	return server, serverSocketPath
}

func newServerClient(t *testing.T, serverSocketPath string) (pb.ServerServiceClient, *grpc.ClientConn) {
	connOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
		grpc.WithPerRPCCredentials(gitalyauth.RPCCredentials(testhelper.RepositoryAuthToken)),
	}
	conn, err := grpc.Dial(serverSocketPath, connOpts...)
	if err != nil {
		t.Fatal(err)
	}

	return pb.NewServerServiceClient(conn), conn
}
