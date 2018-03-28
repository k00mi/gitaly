package commit

import (
	"gitlab.com/gitlab-org/gitaly/internal/helper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

func (s *server) GetCommitMessages(request *pb.GetCommitMessagesRequest, stream pb.CommitService_GetCommitMessagesServer) error {
	return helper.Unimplemented
}
