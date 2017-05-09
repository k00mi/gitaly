package ssh

import (
	"log"
	"os/exec"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	pbh "gitlab.com/gitlab-org/gitaly-proto/go/helper"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type uploadPackBytesReader struct {
	pb.SSH_SSHUploadPackServer
}

type uploadPackWriter struct {
	pb.SSH_SSHUploadPackServer
}

type uploadPackErrorWriter struct {
	pb.SSH_SSHUploadPackServer
}

func (s *server) SSHUploadPack(stream pb.SSH_SSHUploadPackServer) error {
	req, err := stream.Recv() // First request contains Repository only
	if err != nil {
		return err
	}
	if err = validateUploadPackRequest(req); err != nil {
		return err
	}

	streamBytesReader := uploadPackBytesReader{stream}
	stdin := pbh.NewReceiveReader(streamBytesReader.ReceiveBytes)
	stdout := uploadPackWriter{stream}
	stderr := uploadPackErrorWriter{stream}
	repoPath, err := helper.GetRepoPath(req.Repository)
	if err != nil {
		return err
	}

	log.Printf("PostUploadPack: RepoPath=%q", repoPath)

	osCommand := exec.Command("git", "upload-pack", repoPath)
	cmd, err := helper.NewCommand(osCommand, stdin, stdout, stderr)

	if err != nil {
		return grpc.Errorf(codes.Unavailable, "PostUploadPack: cmd: %v", err)
	}
	defer cmd.Kill()

	if err := cmd.Wait(); err != nil {
		if status, ok := helper.ExitStatus(err); ok {
			log.Printf("Exit Status: %d", status)
			stream.Send(&pb.SSHUploadPackResponse{ExitStatus: &pb.ExitStatus{Value: int32(status)}})
			return nil
		}
		return grpc.Errorf(codes.Unavailable, "PostUploadPack: cmd wait for %v: %v", cmd.Args, err)
	}

	return nil
}

func validateUploadPackRequest(req *pb.SSHUploadPackRequest) error {
	if req.Stdin != nil {
		return grpc.Errorf(codes.InvalidArgument, "PostUploadPack: non-empty stdin")
	}

	return nil
}

func (rw uploadPackWriter) Write(p []byte) (int, error) {
	resp := &pb.SSHUploadPackResponse{Stdout: p}
	if err := rw.Send(resp); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (rw uploadPackErrorWriter) Write(p []byte) (int, error) {
	resp := &pb.SSHUploadPackResponse{Stderr: p}
	if err := rw.Send(resp); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (br uploadPackBytesReader) ReceiveBytes() ([]byte, error) {
	resp, err := br.Recv()
	if err != nil {
		return nil, err
	}

	return resp.GetStdin(), nil
}
