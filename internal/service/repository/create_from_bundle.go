package repository

import (
	"io"
	"os"
	"os/exec"
	"path"

	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/tempdir"
	"gitlab.com/gitlab-org/gitaly/streamio"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) CreateRepositoryFromBundle(stream pb.RepositoryService_CreateRepositoryFromBundleServer) error {
	firstRequest, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Internal, "CreateRepositoryFromBundle: first request failed: %v", err)
	}

	repo := firstRequest.GetRepository()
	if repo == nil {
		return status.Errorf(codes.InvalidArgument, "CreateRepositoryFromBundle: empty Repository")
	}

	firstRead := false
	reader := streamio.NewReader(func() ([]byte, error) {
		if !firstRead {
			firstRead = true
			return firstRequest.GetData(), nil
		}

		request, err := stream.Recv()
		return request.GetData(), err
	})

	ctx := stream.Context()

	tmpDir, err := tempdir.New(ctx, repo)
	if err != nil {
		return status.Errorf(codes.Internal, "CreateRepositoryFromBundle: tmp dir failed: %v", err)
	}

	bundlePath := path.Join(tmpDir, "repo.bundle")
	file, err := os.Create(bundlePath)
	if err != nil {
		return status.Errorf(codes.Internal, "CreateRepositoryFromBundle: new bundle file failed: %v", err)
	}

	_, err = io.Copy(file, reader)
	if err != nil {
		return status.Errorf(codes.Internal, "CreateRepositoryFromBundle: bundle write failed: %v", err)
	}

	repoPath, err := helper.GetPath(repo)
	if err != nil {
		return err
	}

	args := []string{
		"clone",
		"--bare",
		"--",
		bundlePath,
		repoPath,
	}
	cmd, err := command.New(ctx, exec.Command(command.GitPath(), args...), nil, nil, nil)
	if err != nil {
		return status.Errorf(codes.Internal, "CreateRepositoryFromBundle: cmd start failed: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		return status.Errorf(codes.Internal, "CreateRepositoryFromBundle: cmd wait failed: %v", err)
	}

	// CreateRepository is harmless on existing repositories with the side effect that it creates the hook symlink.
	if _, err := s.CreateRepository(ctx, &pb.CreateRepositoryRequest{Repository: repo}); err != nil {
		return status.Errorf(codes.Internal, "CreateRepositoryFromBundle: create hooks failed: %v", err)
	}

	return stream.SendAndClose(&pb.CreateRepositoryFromBundleResponse{})
}
