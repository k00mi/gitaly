package remote

import (
	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
)

func (s *server) ListRemotes(*gitalypb.ListRemotesRequest, gitalypb.RemoteService_ListRemotesServer) error {
	return nil
}
