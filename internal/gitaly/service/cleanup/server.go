package cleanup

import (
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type server struct {
	locator storage.Locator
}

// NewServer creates a new instance of a grpc CleanupServer
func NewServer(locator storage.Locator) gitalypb.CleanupServiceServer {
	return &server{locator: locator}
}
