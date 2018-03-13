package ssh

import (
	"os/exec"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/streamio"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) SSHUploadArchive(stream pb.SSHService_SSHUploadArchiveServer) error {
	grpc_logrus.Extract(stream.Context()).Debug("SSHUploadArchive")

	req, err := stream.Recv() // First request contains Repository only
	if err != nil {
		return err
	}
	if err = validateFirstUploadArchiveRequest(req); err != nil {
		return err
	}

	repoPath, err := helper.GetRepoPath(req.Repository)
	if err != nil {
		return err
	}
	stdin := streamio.NewReader(func() ([]byte, error) {
		request, err := stream.Recv()
		return request.GetStdin(), err
	})
	stdout := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&pb.SSHUploadArchiveResponse{Stdout: p})
	})
	stderr := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&pb.SSHUploadArchiveResponse{Stderr: p})
	})

	osCommand := exec.Command(command.GitPath(), "upload-archive", repoPath)

	cmd, err := command.New(stream.Context(), osCommand, stdin, stdout, stderr)

	if err != nil {
		return status.Errorf(codes.Unavailable, "SSHUploadArchive: cmd: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		if status, ok := command.ExitStatus(err); ok {
			return helper.DecorateError(
				codes.Internal,
				stream.Send(&pb.SSHUploadArchiveResponse{ExitStatus: &pb.ExitStatus{Value: int32(status)}}),
			)
		}
		return status.Errorf(codes.Unavailable, "SSHUploadArchive: %v", err)
	}

	return helper.DecorateError(
		codes.Internal,
		stream.Send(&pb.SSHUploadArchiveResponse{ExitStatus: &pb.ExitStatus{Value: 0}}),
	)
}

func validateFirstUploadArchiveRequest(req *pb.SSHUploadArchiveRequest) error {
	if req.Stdin != nil {
		return status.Errorf(codes.InvalidArgument, "SSHUploadArchive: non-empty stdin")
	}

	return nil
}
