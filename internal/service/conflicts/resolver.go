package conflicts

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (s *server) ListConflictFiles(in *pb.ListConflictFilesRequest, stream pb.ConflictsService_ListConflictFilesServer) error {
	return helper.Unimplemented
}

func (s *server) ResolveConflicts(stream pb.ConflictsService_ResolveConflictsServer) error {
	return helper.Unimplemented
}
