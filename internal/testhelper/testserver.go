package testhelper

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	log "github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitaly/internal/helper/fieldextractors"
	praefectconfig "gitlab.com/gitlab-org/gitaly/internal/praefect/config"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/models"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// NewTestServer instantiates a new TestServer
func NewTestServer(srv *grpc.Server) *TestServer {
	return &TestServer{
		grpcServer: srv,
	}
}

// TestServer wraps a grpc Server and handles automatically putting a praefect in front of a gitaly instance
// if necessary
type TestServer struct {
	grpcServer *grpc.Server
	socket     string
	process    *os.Process
}

// GrpcServer returns the underlying grpc.Server
func (p *TestServer) GrpcServer() *grpc.Server {
	return p.grpcServer
}

// Stop will stop both the grpc server as well as the praefect process
func (p *TestServer) Stop() {
	p.grpcServer.Stop()
	if p.process != nil {
		p.process.Kill()
	}
}

// Socket returns the socket file the test server is listening on
func (p *TestServer) Socket() string {
	return p.socket
}

// Start will start the grpc server as well as spawn a praefect instance if GITALY_TEST_PRAEFECT_BIN is enabled
func (p *TestServer) Start() error {
	gitalyServerSocketPath := GetTemporaryGitalySocketFileName()

	listener, err := net.Listen("unix", gitalyServerSocketPath)
	if err != nil {
		return err
	}

	go p.grpcServer.Serve(listener)

	praefectBinPath, ok := os.LookupEnv("GITALY_TEST_PRAEFECT_BIN")
	if !ok {
		p.socket = gitalyServerSocketPath
		return nil
	}

	tempDir, err := ioutil.TempDir("", "praefect-test-server")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	praefectServerSocketPath := GetTemporaryGitalySocketFileName()

	configFilePath := filepath.Join(tempDir, "config.toml")
	configFile, err := os.Create(configFilePath)
	if err != nil {
		return err
	}
	defer configFile.Close()

	c := praefectconfig.Config{
		SocketPath: praefectServerSocketPath,
		VirtualStorages: []*praefectconfig.VirtualStorage{
			{
				Name: "default",
				Nodes: []*models.Node{
					{
						Storage:        "default",
						Address:        "unix:/" + gitalyServerSocketPath,
						DefaultPrimary: true,
					},
				},
			},
		},
	}

	if err := toml.NewEncoder(configFile).Encode(&c); err != nil {
		return err
	}
	if err = configFile.Sync(); err != nil {
		return err
	}
	configFile.Close()

	cmd := exec.Command(praefectBinPath, "-config", configFilePath)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	p.socket = praefectServerSocketPath

	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()

	conn, err := grpc.Dial("unix://"+praefectServerSocketPath, grpc.WithInsecure())

	if err != nil {
		return fmt.Errorf("dial: %v", err)
	}
	defer conn.Close()

	if err = waitForPraefectStartup(conn); err != nil {
		return err
	}

	p.process = cmd.Process

	return nil
}

func waitForPraefectStartup(conn *grpc.ClientConn) error {
	client := healthpb.NewHealthClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{}, grpc.WaitForReady(true))
	if err != nil {
		return err
	}

	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		return errors.New("server not yet ready to serve")
	}

	return nil
}

// NewServer creates a Server for testing purposes
func NewServer(tb testing.TB, streamInterceptors []grpc.StreamServerInterceptor, unaryInterceptors []grpc.UnaryServerInterceptor) *TestServer {
	logger := NewTestLogger(tb)
	logrusEntry := log.NewEntry(logger).WithField("test", tb.Name())

	ctxTagger := grpc_ctxtags.WithFieldExtractorForInitialReq(fieldextractors.FieldExtractor)
	ctxStreamTagger := grpc_ctxtags.StreamServerInterceptor(ctxTagger)
	ctxUnaryTagger := grpc_ctxtags.UnaryServerInterceptor(ctxTagger)

	streamInterceptors = append([]grpc.StreamServerInterceptor{ctxStreamTagger, grpc_logrus.StreamServerInterceptor(logrusEntry)}, streamInterceptors...)
	unaryInterceptors = append([]grpc.UnaryServerInterceptor{ctxUnaryTagger, grpc_logrus.UnaryServerInterceptor(logrusEntry)}, unaryInterceptors...)

	return NewTestServer(
		grpc.NewServer(
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamInterceptors...)),
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryInterceptors...)),
		))
}
