package cache

import (
	"context"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	diskcache "gitlab.com/gitlab-org/gitaly/internal/cache"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/protoregistry"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc"
)

// Invalidator is able to invalidate parts of the cache pertinent to a
// specific repository. Before a repo mutating operation, StartLease should
// be called. Once the operation is complete, the returned LeaseEnder should
// be invoked to end the lease.
type Invalidator interface {
	StartLease(repo *gitalypb.Repository) (diskcache.LeaseEnder, error)
}

type logger func(format string, args ...interface{})

func methodErrLogger(method string) logger {
	return func(format string, args ...interface{}) {
		countMethodErr(method)
		logrus.WithField("full_method_name", method).Errorf(format, args...)
	}
}

// StreamInvalidator will invalidate any mutating RPC that targets a
// repository in a gRPC stream based RPC
func StreamInvalidator(ci Invalidator, reg *protoregistry.Registry) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		errLogger := methodErrLogger(info.FullMethod)

		mInfo, err := reg.LookupMethod(info.FullMethod)
		countRPCType(mInfo)
		if err != nil {
			errLogger("unable to lookup method information for %+v", info)
		}

		if mInfo.Operation == protoregistry.OpAccessor {
			return handler(srv, ss)
		}

		handler, callback := invalidateCache(ci, mInfo, handler, errLogger)
		peeker := newStreamPeeker(ss, callback)
		return handler(srv, peeker)
	}
}

// UnaryInvalidator will invalidate any mutating RPC that targets a
// repository in a gRPC unary RPC
func UnaryInvalidator(ci Invalidator, reg *protoregistry.Registry) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		errLogger := methodErrLogger(info.FullMethod)

		mInfo, err := reg.LookupMethod(info.FullMethod)
		countRPCType(mInfo)
		if err != nil {
			errLogger(info.FullMethod, "unable to lookup method information: %q", err)
			return handler(ctx, req)
		}

		if mInfo.Operation == protoregistry.OpAccessor {
			return handler(ctx, req)
		}

		pbReq, ok := req.(proto.Message)
		if !ok {
			errLogger("expected protobuf message but got %T", req)
			return handler(ctx, req)
		}

		target, err := mInfo.TargetRepo(pbReq)
		if err != nil {
			errLogger("unable to extract target repo: %q", err)
			return handler(ctx, req)
		}

		le, err := ci.StartLease(target)
		if err != nil {
			errLogger("unable to start lease: %q", err)
			return handler(ctx, req)
		}

		// wrap the handler to ensure the lease is always ended
		return func() (resp interface{}, err error) {
			defer func() {
				if err := le.EndLease(ctx); err != nil {
					errLogger("unable to end lease: %q", err)
				}
			}()
			return handler(ctx, req)
		}()
	}
}

type recvMsgCallback func(interface{}, error)

func invalidateCache(ci Invalidator, mInfo protoregistry.MethodInfo, handler grpc.StreamHandler, errLogger logger) (grpc.StreamHandler, recvMsgCallback) {
	var le struct {
		sync.RWMutex
		diskcache.LeaseEnder
	}

	// ensures that the lease ender is invoked after the original handler
	wrappedHandler := func(srv interface{}, stream grpc.ServerStream) error {
		defer func() {
			le.RLock()
			defer le.RUnlock()

			if le.LeaseEnder == nil {
				return
			}
			if err := le.EndLease(stream.Context()); err != nil {
				errLogger("unable to end lease: %q", err)
			}
		}()
		return handler(srv, stream)
	}

	// starts the cache lease and sets the lease ender iff the request's target
	// repository can be determined from the first request message
	peekerCallback := func(firstReq interface{}, err error) {
		if err != nil {
			errLogger("peeker received an error: %q", err)
		}

		pbFirstReq, ok := firstReq.(proto.Message)
		if !ok {
			errLogger("cache invalidation expected protobuf request, but got %T", firstReq)
		}

		target, err := mInfo.TargetRepo(pbFirstReq)
		if err != nil {
			errLogger("could not extract target repo: %q", err)
		}

		le.Lock()
		defer le.Unlock()

		le.LeaseEnder, err = ci.StartLease(target)
		if err != nil {
			errLogger("could not start lease: %q", err)
		}
	}

	return wrappedHandler, peekerCallback
}

// streamPeeker allows a stream interceptor to insert peeking logic to perform
// an action when the first RecvMsg
type streamPeeker struct {
	grpc.ServerStream

	// onFirstRecvCallback is called the first time the server stream's RecvMsg
	// is invoked. It passes the results of the stream's RecvMsg as the
	// callback's parameters.
	onFirstRecvOnce     sync.Once
	onFirstRecvCallback recvMsgCallback
}

// newStreamPeeker returns a wrapped stream that allows a callback to be called
// on the first invocation of RecvMsg.
func newStreamPeeker(stream grpc.ServerStream, callback recvMsgCallback) grpc.ServerStream {
	return &streamPeeker{
		ServerStream:        stream,
		onFirstRecvCallback: callback,
	}
}

// RecvMsg overrides the embedded grpc.ServerStream's method of the same name so
// that the callback is called on the first call.
func (sp *streamPeeker) RecvMsg(m interface{}) error {
	err := sp.ServerStream.RecvMsg(m)
	sp.onFirstRecvOnce.Do(func() { sp.onFirstRecvCallback(m, err) })
	return err
}
