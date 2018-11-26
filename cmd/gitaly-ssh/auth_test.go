package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
	"gitlab.com/gitlab-org/gitaly/internal/server"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"google.golang.org/grpc"
)

func buildGitalySSH(t *testing.T) {
	// Build the test-binary that we need
	os.Remove("gitaly-ssh")
	testhelper.MustRunCommand(nil, nil, "go", "build", "gitlab.com/gitlab-org/gitaly/cmd/gitaly-ssh")
}

func TestConnectivity(t *testing.T) {
	buildGitalySSH(t)
	testRepo := testhelper.TestRepository()

	cwd, err := os.Getwd()
	require.NoError(t, err)
	gitalySSHPath := path.Join(cwd, "gitaly-ssh")

	socketPath := testhelper.GetTemporaryGitalySocketFileName()

	tcpServer, tcpPort := runServer(t, server.New, "tcp", "localhost:0")
	defer tcpServer.Stop()

	unixServer, _ := runServer(t, server.New, "unix", socketPath)
	defer unixServer.Stop()

	testCases := []struct {
		addr string
	}{
		{
			addr: fmt.Sprintf("tcp://localhost:%d", tcpPort),
		},
		{
			addr: fmt.Sprintf("unix://%s", socketPath),
		},
	}

	pbMarshaler := &jsonpb.Marshaler{}
	payload, err := pbMarshaler.MarshalToString(&gitalypb.SSHUploadPackRequest{
		Repository: testRepo,
	})

	require.NoError(t, err)
	for _, testcase := range testCases {
		cmd := exec.Command("git", "ls-remote", "git@localhost:test/test.git", "refs/heads/master")

		cmd.Env = []string{
			fmt.Sprintf("GITALY_PAYLOAD=%s", payload),
			fmt.Sprintf("GITALY_ADDRESS=%s", testcase.addr),
			fmt.Sprintf("PATH=.:%s", os.Getenv("PATH")),
			fmt.Sprintf("GIT_SSH_COMMAND=%s upload-pack", gitalySSHPath),
		}

		output, err := cmd.Output()

		require.NoError(t, err)
		require.True(t, strings.HasSuffix(strings.TrimSpace(string(output)), "refs/heads/master"))
	}
}

func runServer(t *testing.T, newServer func(rubyServer *rubyserver.Server) *grpc.Server, connectionType string, addr string) (*grpc.Server, int) {
	srv := newServer(nil)

	listener, err := net.Listen(connectionType, addr)
	require.NoError(t, err)

	go srv.Serve(listener)

	port := 0
	if connectionType != "unix" {
		addrSplit := strings.Split(listener.Addr().String(), ":")
		portString := addrSplit[len(addrSplit)-1]

		port, err = strconv.Atoi(portString)
		require.NoError(t, err)
	}

	return srv, port
}
