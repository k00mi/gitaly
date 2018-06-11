package repository

import (
	"os"
	"path"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/archive"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/streamio"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) BackupCustomHooks(in *pb.BackupCustomHooksRequest, stream pb.RepositoryService_BackupCustomHooksServer) error {
	repoPath, err := helper.GetPath(in.Repository)
	if err != nil {
		return status.Errorf(codes.Internal, "BackupCustomHooks: getting repo path failed %v", err)
	}

	writer := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&pb.BackupCustomHooksResponse{Data: p})
	})

	if _, err := os.Lstat(path.Join(repoPath, "custom_hooks")); os.IsNotExist(err) {
		return nil
	}

	builder := archive.NewTarBuilder(repoPath, writer)
	builder.RecursiveDirIfExist("custom_hooks")
	if err = builder.Close(); err != nil {
		return status.Errorf(codes.Internal, "BackupCustomHooks: adding custom_hooks failed: %v", err)
	}

	return nil
}
