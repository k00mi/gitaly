package loghandler

import (
	"time"

	raven "github.com/getsentry/raven-go"
	log "github.com/sirupsen/logrus"

	"math"

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

func roundPositive(value float64) float64 {
	return math.Floor(value + 0.5)
}

// durationInSecondsRoundedToMilliseconds returns a duration, in seconds with a maximum resolution of a microsecond
func durationInSecondsRoundedToMilliseconds(d time.Duration) float64 {
	return roundPositive(d.Seconds()*1e6) / 1e6
}

func logGrpcError(method string, err error) {
	grpcErrorCode := grpc.Code(err)

	if grpcErrorCode == codes.OK {
		return
	}

	raven.CaptureError(err, map[string]string{
		"grpcMethod": method,
		"code":       grpcErrorCode.String(),
	}, nil)

	log.WithFields(log.Fields{
		"method": method,
		"code":   grpcErrorCode.String(),
		"error":  err,
	}).Error("grpc error response")
}

func logRequest(method string, start time.Time, err error) {
	duration := durationInSecondsRoundedToMilliseconds(time.Since(start))
	fields := log.Fields{
		"method":   method,
		"duration": duration,
	}

	if err != nil {
		grpcErrorCode := grpc.Code(err)
		fields["error"] = err
		fields["code"] = grpcErrorCode.String()

		logGrpcError(method, err)
	}

	log.WithFields(fields).Info("access")
}
