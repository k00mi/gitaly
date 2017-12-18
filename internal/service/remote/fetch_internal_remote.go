package remote

import (
	"golang.org/x/net/context"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

// FetchInternalRemote fetches another Gitaly repository set as a remote
func (s *server) FetchInternalRemote(context.Context, *pb.FetchInternalRemoteRequest) (*pb.FetchInternalRemoteResponse, error) {
	return nil, helper.Unimplemented
}
