package ref

import (
	"fmt"
	"strings"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/git/catfile"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) FindAllRemoteBranches(req *pb.FindAllRemoteBranchesRequest, stream pb.RefService_FindAllRemoteBranchesServer) error {
	if err := validateFindAllRemoteBranchesRequest(req); err != nil {
		return status.Errorf(codes.InvalidArgument, "FindAllRemoteBranches: %v", err)
	}

	args := []string{
		"--format=" + strings.Join(localBranchFormatFields, "%00"),
	}

	patterns := []string{"refs/remotes/" + req.GetRemoteName()}

	ctx := stream.Context()
	c, err := catfile.New(ctx, req.GetRepository())
	if err != nil {
		return err
	}

	opts := &findRefsOpts{
		cmdArgs: args,
	}
	writer := newFindAllRemoteBranchesWriter(stream, c)

	return findRefs(ctx, writer, req.GetRepository(), patterns, opts)
}

func validateFindAllRemoteBranchesRequest(req *pb.FindAllRemoteBranchesRequest) error {
	if req.GetRepository() == nil {
		return fmt.Errorf("empty Repository")
	}

	if len(req.GetRemoteName()) == 0 {
		return fmt.Errorf("empty RemoteName")
	}

	return nil
}
