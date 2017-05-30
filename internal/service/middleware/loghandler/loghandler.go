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
	logRequest(ctx, info.FullMethod, start, err)
	return resp, err
}

// StreamLogHandler handles access times and errors for stream RPC's
func StreamLogHandler(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	err := handler(srv, stream)
	logRequest(stream.Context(), info.FullMethod, start, err)
	return err
}

func roundPositive(value float64) float64 {
	return math.Floor(value + 0.5)
}

// durationInSecondsRoundedToMilliseconds returns a duration, in seconds with a maximum resolution of a microsecond
func durationInSecondsRoundedToMilliseconds(d time.Duration) float64 {
	return roundPositive(d.Seconds()*1e6) / 1e6
}

func logGrpcError(ctx context.Context, method string, err error) {
	grpcErrorCode := grpc.Code(err)

	loggerWithFields := log.WithFields(log.Fields{
		"method": method,
		"code":   grpcErrorCode.String(),
		"error":  err,
	})

	switch grpcErrorCode {

	// This probably won't happen
	case codes.OK:
		return

	// Things we consider to be warnings: ie problems with the client
	// these should not be logged in sentry
	case codes.Canceled:
	case codes.InvalidArgument:
	case codes.NotFound:
	case codes.AlreadyExists:
	case codes.PermissionDenied:
	case codes.Unauthenticated:
	case codes.FailedPrecondition:
	case codes.OutOfRange:
		loggerWithFields.Warn("grpc error response")

	// Everything else we consider to be problems with the server
	// log these as Errors and also log them to sentry
	default:
		raven.CaptureError(err, map[string]string{
			"grpcMethod": method,
			"code":       grpcErrorCode.String(),
		}, nil)

		loggerWithFields.Error("grpc error response")

	}

}

func logRequest(ctx context.Context, method string, start time.Time, err error) {
	duration := durationInSecondsRoundedToMilliseconds(time.Since(start))
	fields := log.Fields{
		"method":   method,
		"duration": duration,
	}

	if err != nil {
		grpcErrorCode := grpc.Code(err)
		fields["error"] = err
		fields["code"] = grpcErrorCode.String()

		logGrpcError(ctx, method, err)
	}

	log.WithFields(fields).Info("access")
}
