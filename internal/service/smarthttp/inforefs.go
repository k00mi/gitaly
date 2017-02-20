package smarthttp

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type infoRefsResponseWriter struct {
	infoRefsResponseServer
}

type infoRefsResponseServer interface {
	Send(*pb.InfoRefsResponse) error
}

func (w infoRefsResponseWriter) Write(p []byte) (int, error) {
	if err := w.Send(&pb.InfoRefsResponse{Data: p}); err != nil {
		return 0, err
	}

	return len(p), nil
}

func (s *server) InfoRefsUploadPack(in *pb.InfoRefsRequest, stream pb.SmartHTTP_InfoRefsUploadPackServer) error {
	return handleInfoRefs("upload-pack", in.Repository, infoRefsResponseWriter{stream})
}

func (s *server) InfoRefsReceivePack(in *pb.InfoRefsRequest, stream pb.SmartHTTP_InfoRefsReceivePackServer) error {
	return handleInfoRefs("receive-pack", in.Repository, infoRefsResponseWriter{stream})
}

func handleInfoRefs(service string, repo *pb.Repository, w io.Writer) error {
	if repo == nil {
		return grpc.Errorf(codes.InvalidArgument, "GetInfoRefs: repo argument is missing")
	}
	cmd := gitCommand("", "git", service, "--stateless-rpc", "--advertise-refs", repo.Path)

	log.Printf("handleInfoRefs: service=%q RepoPath=%q", service, repo.Path)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return grpc.Errorf(codes.Internal, "GetInfoRefs: stdout: %v", err)
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return grpc.Errorf(codes.Internal, "GetInfoRefs: start %v: %v", cmd.Args, err)
	}
	defer helper.CleanUpProcessGroup(cmd) // Ensure brute force subprocess clean-up

	if err := pktLine(w, fmt.Sprintf("# service=git-%s\n", service)); err != nil {
		return grpc.Errorf(codes.Internal, "GetInfoRefs: pktLine: %v", err)
	}

	if err := pktFlush(w); err != nil {
		return grpc.Errorf(codes.Internal, "GetInfoRefs: pktFlush: %v", err)
	}

	if _, err := io.Copy(w, stdout); err != nil {
		return grpc.Errorf(codes.Internal, "GetInfoRefs: copy output of %v: %v", cmd.Args, err)
	}

	if err := cmd.Wait(); err != nil {
		return grpc.Errorf(codes.Internal, "GetInfoRefs: wait for %v: %v", cmd.Args, err)
	}

	return nil
}

func gitCommand(glID string, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	// Start the command in its own process group (nice for signalling)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Explicitly set the environment for the Git command
	cmd.Env = []string{
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("LD_LIBRARY_PATH=%s", os.Getenv("LD_LIBRARY_PATH")),
		fmt.Sprintf("GL_ID=%s", glID),
		fmt.Sprintf("GL_PROTOCOL=http"),
	}
	// If we don't do something with cmd.Stderr, Git errors will be lost
	cmd.Stderr = os.Stderr
	return cmd
}

func pktLine(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "%04x%s", len(s)+4, s)
	return err
}

func pktFlush(w io.Writer) error {
	_, err := fmt.Fprint(w, "0000")
	return err
}
