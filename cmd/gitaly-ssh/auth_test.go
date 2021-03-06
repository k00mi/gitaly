package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/hook"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/rubyserver"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/server"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc"
)

//go:generate openssl req -newkey rsa:4096 -new -nodes -x509 -days 3650 -out testdata/certs/gitalycert.pem -keyout testdata/gitalykey.pem -subj "/C=US/ST=California/L=San Francisco/O=GitLab/OU=GitLab-Shell/CN=localhost" -addext "subjectAltName = IP:127.0.0.1, DNS:localhost"
func TestConnectivity(t *testing.T) {
	// regenerate the test cert and key via `go generate`
	config.Config.TLS = config.TLS{
		CertPath: "testdata/certs/gitalycert.pem",
		KeyPath:  "testdata/gitalykey.pem",
	}

	cwd, err := os.Getwd()
	require.NoError(t, err)

	certPoolPath := filepath.Join(cwd, "testdata", "certs")

	testRepo, _, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	socketPath := testhelper.GetTemporaryGitalySocketFileName()

	tempDir, cleanup := testhelper.TempDir(t)
	defer cleanup()

	relativeSocketPath, err := filepath.Rel(cwd, filepath.Join(tempDir, "gitaly.socket"))
	require.NoError(t, err)

	require.NoError(t, os.RemoveAll(relativeSocketPath))
	require.NoError(t, os.Symlink(socketPath, relativeSocketPath))

	tcpPort, cleanTCP := runServer(t, server.NewInsecure, config.Config, "tcp", "localhost:0")
	defer cleanTCP()

	tlsPort, cleanTLS := runServer(t, server.NewSecure, config.Config, "tcp", "localhost:0")
	defer cleanTLS()

	_, cleanUnix := runServer(t, server.NewInsecure, config.Config, "unix", socketPath)
	defer cleanUnix()

	testCases := []struct {
		name  string
		addr  string
		proxy bool
	}{
		{
			name: "tcp",
			addr: fmt.Sprintf("tcp://localhost:%d", tcpPort),
		},
		{
			name: "unix absolute",
			addr: fmt.Sprintf("unix:%s", socketPath),
		},
		{
			name:  "unix abs with proxy",
			addr:  fmt.Sprintf("unix:%s", socketPath),
			proxy: true,
		},
		{
			name: "unix relative",
			addr: fmt.Sprintf("unix:%s", relativeSocketPath),
		},
		{
			name:  "unix relative with proxy",
			addr:  fmt.Sprintf("unix:%s", relativeSocketPath),
			proxy: true,
		},

		{
			name: "tls",
			addr: fmt.Sprintf("tls://localhost:%d", tlsPort),
		},
	}

	pbMarshaler := &jsonpb.Marshaler{}
	payload, err := pbMarshaler.MarshalToString(&gitalypb.SSHUploadPackRequest{
		Repository: testRepo,
	})

	require.NoError(t, err)
	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			cmd := exec.Command(config.Config.Git.BinPath, "ls-remote", "git@localhost:test/test.git", "refs/heads/master")
			cmd.Stderr = os.Stderr
			cmd.Env = []string{
				fmt.Sprintf("GITALY_PAYLOAD=%s", payload),
				fmt.Sprintf("GITALY_ADDRESS=%s", testcase.addr),
				fmt.Sprintf("GITALY_WD=%s", cwd),
				fmt.Sprintf("PATH=.:%s", os.Getenv("PATH")),
				fmt.Sprintf("GIT_SSH_COMMAND=%s upload-pack", gitalySSHPath),
				fmt.Sprintf("SSL_CERT_DIR=%s", certPoolPath),
			}

			if testcase.proxy {
				cmd.Env = append(cmd.Env,
					"http_proxy=http://invalid:1234",
					"https_proxy=https://invalid:1234",
				)
			}

			output, err := cmd.Output()

			require.NoError(t, err, "git ls-remote exit status")
			require.True(t, strings.HasSuffix(strings.TrimSpace(string(output)), "refs/heads/master"))
		})
	}
}

func runServer(t *testing.T, newServer func(rubyServer *rubyserver.Server, hookManager hook.Manager, cfg config.Cfg, conns *client.Pool) *grpc.Server, cfg config.Cfg, connectionType string, addr string) (int, func()) {
	conns := client.NewPool()
	srv := newServer(nil, hook.NewManager(config.NewLocator(cfg), hook.GitlabAPIStub, cfg), cfg, conns)

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

	return port, func() {
		conns.Close()
		srv.Stop()
	}
}
