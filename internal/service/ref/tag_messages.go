package ref

import (
	"gitlab.com/gitlab-org/gitaly/internal/helper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

func (s *server) GetTagMessages(request *pb.GetTagMessagesRequest, stream pb.RefService_GetTagMessagesServer) error {
	return helper.Unimplemented
}
