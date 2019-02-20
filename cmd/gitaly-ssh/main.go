package main

import (
	"context"
	"fmt"
	"log"
	"os"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	gitalyauth "gitlab.com/gitlab-org/gitaly/auth"
	"gitlab.com/gitlab-org/gitaly/client"
	grpccorrelation "gitlab.com/gitlab-org/labkit/correlation/grpc"
	"gitlab.com/gitlab-org/labkit/tracing"
	grpctracing "gitlab.com/gitlab-org/labkit/tracing/grpc"
	"google.golang.org/grpc"
)

type packFn func(_ context.Context, _ *grpc.ClientConn, _ string) (int32, error)

// GITALY_ADDRESS="tcp://1.2.3.4:9999" or "unix:/var/run/gitaly.sock"
// GITALY_TOKEN="foobar1234"
// GITALY_PAYLOAD="{repo...}"
// GITALY_WD="/path/to/working-directory"
// gitaly-ssh upload-pack <git-garbage-x2>
func main() {
	// < 4 since git throws on 2x garbage here
	if n := len(os.Args); n < 4 {
		// TODO: Errors needs to be sent back some other way... pipes?
		log.Fatalf("invalid number of arguments, expected at least 1, got %d", n-1)
	}

	command := os.Args[1]
	var packer packFn
	switch command {
	case "upload-pack":
		packer = uploadPack
	case "receive-pack":
		packer = receivePack
	case "upload-archive":
		packer = uploadArchive
	default:
		log.Fatalf("invalid pack command: %q", command)
	}

	gitalyWorkingDir := os.Getenv("GITALY_WD")
	gitalyAddress := os.Getenv("GITALY_ADDRESS")
	gitalyPayload := os.Getenv("GITALY_PAYLOAD")

	code, err := run(packer, gitalyWorkingDir, gitalyAddress, gitalyPayload)
	if err != nil {
		log.Printf("%s: %v", command, err)
	}

	os.Exit(code)
}

func run(packer packFn, gitalyWorkingDir string, gitalyAddress string, gitalyPayload string) (int, error) {
	// Configure distributed tracing
	closer := tracing.Initialize(tracing.WithServiceName("gitaly-ssh"))
	defer closer.Close()

	ctx, finished := tracing.ExtractFromEnv(context.Background())
	defer finished()

	if gitalyWorkingDir != "" {
		if err := os.Chdir(gitalyWorkingDir); err != nil {
			return 1, fmt.Errorf("unable to chdir to %v", gitalyWorkingDir)
		}
	}

	conn, err := getConnection(gitalyAddress)
	if err != nil {
		return 1, err
	}
	defer conn.Close()

	code, err := packer(ctx, conn, gitalyPayload)
	if err != nil {
		return 1, err
	}

	return int(code), nil
}

func getConnection(url string) (*grpc.ClientConn, error) {
	if url == "" {
		return nil, fmt.Errorf("gitaly address can not be empty")
	}

	return client.Dial(url, dialOpts())
}

func dialOpts() []grpc.DialOption {
	connOpts := client.DefaultDialOpts
	if token := os.Getenv("GITALY_TOKEN"); token != "" {
		connOpts = append(connOpts, grpc.WithPerRPCCredentials(gitalyauth.RPCCredentials(token)))
	}

	// Add grpc client interceptors
	connOpts = append(connOpts, grpc.WithStreamInterceptor(
		grpc_middleware.ChainStreamClient(
			grpctracing.StreamClientTracingInterceptor(),         // Tracing
			grpccorrelation.StreamClientCorrelationInterceptor(), // Correlation
		)),

		grpc.WithUnaryInterceptor(
			grpc_middleware.ChainUnaryClient(
				grpctracing.UnaryClientTracingInterceptor(),         // Tracing
				grpccorrelation.UnaryClientCorrelationInterceptor(), // Correlation
			)))

	return connOpts
}
