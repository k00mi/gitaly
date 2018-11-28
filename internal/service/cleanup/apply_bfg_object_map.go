package cleanup

import (
	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"

	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (s *server) ApplyBfgObjectMap(gitalypb.CleanupService_ApplyBfgObjectMapServer) error {
	return helper.Unimplemented
}
