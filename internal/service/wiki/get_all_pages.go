package wiki

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (s *server) WikiGetAllPages(request *pb.WikiGetAllPagesRequest, stream pb.WikiService_WikiGetAllPagesServer) error {
	return helper.Unimplemented
}
