package ssh

import (
	"fmt"
	"log"
	"os/exec"

	"gitlab.com/gitlab-org/gitaly/internal/helper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	pbh "gitlab.com/gitlab-org/gitaly-proto/go/helper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type receivePackBytesReader struct {
	pb.SSH_SSHReceivePackServer
}

type receivePackWriter struct {
	pb.SSH_SSHReceivePackServer
}

type receivePackErrorWriter struct {
	pb.SSH_SSHReceivePackServer
}

func (s *server) SSHReceivePack(stream pb.SSH_SSHReceivePackServer) error {
	req, err := stream.Recv() // First request contains only Repository and GlId
	if err != nil {
		return err
	}
	if err = validateReceivePackRequest(req); err != nil {
		return err
	}

	streamBytesReader := receivePackBytesReader{stream}
	stdin := pbh.NewReceiveReader(streamBytesReader.ReceiveBytes)
	stdout := receivePackWriter{stream}
	stderr := receivePackErrorWriter{stream}
	env := []string{
		fmt.Sprintf("GL_ID=%s", req.GlId),
		"GL_PROTOCOL=ssh",
	}

	repoPath, err := helper.GetRepoPath(req.Repository)
	if err != nil {
		return err
	}

	log.Printf("PostReceivePack: RepoPath=%q GlID=%q", repoPath, req.GlId)

	osCommand := exec.Command("git-receive-pack", repoPath)
	cmd, err := helper.NewCommand(osCommand, stdin, stdout, stderr, env...)

	if err != nil {
		return grpc.Errorf(codes.Unavailable, "PostReceivePack: cmd: %v", err)
	}
	defer cmd.Kill()

	if err := cmd.Wait(); err != nil {
		if status, ok := helper.ExitStatus(err); ok {
			log.Printf("Exit Status: %d", status)
			stream.Send(&pb.SSHReceivePackResponse{ExitStatus: &pb.ExitStatus{Value: int32(status)}})
			return nil
		}
		return grpc.Errorf(codes.Unavailable, "PostReceivePack: cmd wait for %v: %v", cmd.Args, err)
	}

	return nil
}

func validateReceivePackRequest(req *pb.SSHReceivePackRequest) error {
	if req.GlId == "" {
		return grpc.Errorf(codes.InvalidArgument, "PostReceivePack: empty GlId")
	}
	if req.Stdin != nil {
		return grpc.Errorf(codes.InvalidArgument, "PostReceivePack: non-empty data")
	}

	return nil
}

func (rw receivePackWriter) Write(p []byte) (int, error) {
	resp := &pb.SSHReceivePackResponse{Stdout: p}
	if err := rw.Send(resp); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (rw receivePackErrorWriter) Write(p []byte) (int, error) {
	resp := &pb.SSHReceivePackResponse{Stderr: p}
	if err := rw.Send(resp); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (br receivePackBytesReader) ReceiveBytes() ([]byte, error) {
	resp, err := br.Recv()
	if err != nil {
		return nil, err
	}

	return resp.GetStdin(), nil
}
