package client

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	gitaly_x509 "gitlab.com/gitlab-org/gitaly/internal/x509"
	grpccorrelation "gitlab.com/gitlab-org/labkit/correlation/grpc"
	grpctracing "gitlab.com/gitlab-org/labkit/tracing/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

// DefaultDialOpts hold the default DialOptions for connection to Gitaly over UNIX-socket
var DefaultDialOpts = []grpc.DialOption{}

type connectionType int

const (
	invalidConnection connectionType = iota
	tcpConnection
	tlsConnection
	unixConnection
)

func DialContext(ctx context.Context, rawAddress string, connOpts []grpc.DialOption) (*grpc.ClientConn, error) {
	var canonicalAddress string
	var err error

	switch getConnectionType(rawAddress) {
	case invalidConnection:
		return nil, fmt.Errorf("invalid connection string: %q", rawAddress)

	case tlsConnection:
		canonicalAddress, err = extractHostFromRemoteURL(rawAddress) // Ensure the form: "host:port" ...
		if err != nil {
			return nil, fmt.Errorf("failed to extract host for 'tls' connection: %w", err)
		}

		certPool, err := gitaly_x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to get system certificat pool for 'tls' connection: %w", err)
		}

		creds := credentials.NewClientTLSFromCert(certPool, "")
		connOpts = append(connOpts, grpc.WithTransportCredentials(creds))

	case tcpConnection:
		canonicalAddress, err = extractHostFromRemoteURL(rawAddress) // Ensure the form: "host:port" ...
		if err != nil {
			return nil, fmt.Errorf("failed to extract host for 'tcp' connection: %w", err)
		}
		connOpts = append(connOpts, grpc.WithInsecure())

	case unixConnection:
		canonicalAddress = rawAddress // This will be overridden by the custom dialer...
		connOpts = append(
			connOpts,
			grpc.WithInsecure(),
			// Use a custom dialer to ensure that we don't experience
			// issues in environments that have proxy configurations
			// https://gitlab.com/gitlab-org/gitaly/merge_requests/1072#note_140408512
			grpc.WithContextDialer(func(ctx context.Context, addr string) (conn net.Conn, err error) {
				path, err := extractPathFromSocketURL(addr)
				if err != nil {
					return nil, fmt.Errorf("failed to extract host for 'unix' connection: %w", err)
				}

				d := net.Dialer{}
				return d.DialContext(ctx, "unix", path)
			}),
		)
	}

	connOpts = append(connOpts,
		// grpc.KeepaliveParams must be specified at least as large as what is allowed by the
		// server-side grpc.KeepaliveEnforcementPolicy
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithChainUnaryInterceptor(
			grpctracing.UnaryClientTracingInterceptor(),
			grpccorrelation.UnaryClientCorrelationInterceptor(),
		),
		grpc.WithChainStreamInterceptor(
			grpctracing.StreamClientTracingInterceptor(),
			grpccorrelation.StreamClientCorrelationInterceptor(),
		),
	)

	conn, err := grpc.DialContext(ctx, canonicalAddress, connOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %q connection: %w", canonicalAddress, err)
	}

	return conn, nil
}

func Dial(rawAddress string, connOpts []grpc.DialOption) (*grpc.ClientConn, error) {
	return DialContext(context.Background(), rawAddress, connOpts)
}

func getConnectionType(rawAddress string) connectionType {
	u, err := url.Parse(rawAddress)
	if err != nil {
		return invalidConnection
	}

	switch u.Scheme {
	case "tls":
		return tlsConnection
	case "unix":
		return unixConnection
	case "tcp":
		return tcpConnection
	default:
		return invalidConnection
	}
}

// FailOnNonTempDialError helps to identify if remote listener is ready to accept new connections.
func FailOnNonTempDialError() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithBlock(),
		grpc.FailOnNonTempDialError(true),
	}
}

// HealthCheckDialer uses provided dialer as an actual dialer, but issues a health check request to the remote
// to verify the connection was set properly and could be used with no issues.
func HealthCheckDialer(base Dialer) Dialer {
	return func(ctx context.Context, address string, dialOptions []grpc.DialOption) (*grpc.ClientConn, error) {
		cc, err := base(ctx, address, dialOptions)
		if err != nil {
			return nil, err
		}

		if _, err := healthpb.NewHealthClient(cc).Check(ctx, &healthpb.HealthCheckRequest{}); err != nil {
			cc.Close()
			return nil, err
		}

		return cc, nil
	}
}
