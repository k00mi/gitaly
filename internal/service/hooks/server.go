package hook

import (
	"sync"

	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc"
)

type server struct {
	mutex            sync.RWMutex
	praefectConnPool map[string]*grpc.ClientConn
}

// NewServer creates a new instance of a gRPC namespace server
func NewServer() gitalypb.HookServiceServer {
	return &server{
		praefectConnPool: make(map[string]*grpc.ClientConn),
	}
}
