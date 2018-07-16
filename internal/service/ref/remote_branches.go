package ref

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (s *server) FindAllRemoteBranches(in *pb.FindAllRemoteBranchesRequest, stream pb.RefService_FindAllRemoteBranchesServer) error {
	return helper.Unimplemented
}
