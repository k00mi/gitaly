package info

import (
	"gitlab.com/gitlab-org/gitaly/internal/praefect/nodes"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

// Server is a InfoService server
type Server struct {
	gitalypb.UnimplementedPraefectInfoServiceServer
	nodeMgr nodes.Manager
}

// NewServer creates a new instance of a grpc InfoServiceServer
func NewServer(nodeMgr nodes.Manager) gitalypb.PraefectInfoServiceServer {
	s := &Server{
		nodeMgr: nodeMgr,
	}

	return s
}
