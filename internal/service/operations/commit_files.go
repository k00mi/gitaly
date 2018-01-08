package operations

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (*server) UserCommitFiles(pb.OperationService_UserCommitFilesServer) error {
	return helper.Unimplemented
}
