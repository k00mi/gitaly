package loghandler

import (
	"time"

	raven "github.com/getsentry/raven-go"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// UnaryLogHandler handles access times and errors for unary RPC's
func UnaryLogHandler(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	logRequest(info.FullMethod, start, err)
	return resp, err
}

// StreamLogHandler handles access times and errors for stream RPC's
func StreamLogHandler(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	err := handler(srv, stream)
	logRequest(info.FullMethod, start, err)
	return err
}

func logRequest(method string, start time.Time, err error) {
	if err == nil {
		return
	}

	grpcErrorCode := grpc.Code(err)

	if grpcErrorCode == codes.OK {
		return
	}

	raven.CaptureError(err, map[string]string{
		"grpcMethod": method,
		"code":       grpcErrorCode.String(),
	}, nil)

}
