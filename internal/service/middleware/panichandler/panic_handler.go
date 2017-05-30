package panichandler

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var _ grpc.UnaryServerInterceptor = UnaryPanicHandler
var _ grpc.StreamServerInterceptor = StreamPanicHandler

func toPanicError(grpcMethodName string, r interface{}) error {
	return grpc.Errorf(codes.Internal, "panic: %v", r)
}

// UnaryPanicHandler handles request-response panics
func UnaryPanicHandler(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer handleCrash(ctx, info.FullMethod, func(ctx context.Context, grpcMethodName string, r interface{}) {
		err = toPanicError(grpcMethodName, r)
	})

	return handler(ctx, req)
}

// StreamPanicHandler handles stream panics
func StreamPanicHandler(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer handleCrash(stream.Context(), info.FullMethod, func(ctx context.Context, grpcMethodName string, r interface{}) {
		err = toPanicError(grpcMethodName, r)
	})

	return handler(srv, stream)
}

var additionalHandlers []func(context.Context, string, interface{})

// InstallPanicHandler installs additional crash handles for dealing with a panic
func InstallPanicHandler(handler func(context.Context, string, interface{})) {
	additionalHandlers = append(additionalHandlers, handler)
}

func handleCrash(ctx context.Context, grpcMethodName string, handler func(context.Context, string, interface{})) {
	if r := recover(); r != nil {
		handler(ctx, grpcMethodName, r)

		if additionalHandlers != nil {
			for _, fn := range additionalHandlers {
				fn(ctx, grpcMethodName, r)
			}
		}
	}
}
