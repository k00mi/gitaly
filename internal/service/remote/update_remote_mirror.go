package remote

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (*server) UpdateRemoteMirror(pb.RemoteService_UpdateRemoteMirrorServer) error {
	return helper.Unimplemented
}
