package conflicts

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
)

type server struct {
	*rubyserver.Server
}

// NewServer creates a new instance of a grpc ConflictsServer
func NewServer(rs *rubyserver.Server) pb.ConflictsServiceServer {
	return &server{rs}
}
