package wiki

import (
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

type server struct {
	*rubyserver.Server
}

// NewServer creates a new instance of a grpc WikiServiceServer
func NewServer(rs *rubyserver.Server) pb.WikiServiceServer {
	return &server{rs}
}

func (s *server) WikiGetPageVersions(_ *pb.WikiGetPageVersionsRequest, _ pb.WikiService_WikiGetPageVersionsServer) error {
	return helper.Unimplemented
}
