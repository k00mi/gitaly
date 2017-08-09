package diff

import (
	"bufio"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/helper/lines"
)

func (server) CommitPatch(in *pb.CommitPatchRequest, stream pb.DiffService_CommitPatchServer) error {
	repoPath, err := helper.GetRepoPath(in.Repository)
	if err != nil {
		return err
	}

	args := []string{"-C", repoPath, "show", "--format=%N", string(in.GetRevision())}
	cmd, err := helper.GitCommandReader(stream.Context(), args...)
	if err != nil {
		return grpc.Errorf(codes.Internal, "CommitPatch: cmd: %v", err)
	}

	reader := bufio.NewReader(cmd)
	buf := make([]byte, lines.MaxMsgSize)

	// Discard leading `\n`s
	if _, err := reader.Discard(2); err != nil {
		return err
	}

	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return grpc.Errorf(codes.Internal, "CommitPatch: read: %v", err)
		}
		if n == 0 {
			break
		}
		if err := stream.Send(&pb.CommitPatchResponse{Data: buf[0:n]}); err != nil {
			return err
		}
	}

	return nil
}
