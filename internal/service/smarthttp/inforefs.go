package smarthttp

import (
	"context"
	"fmt"
	"io"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	log "github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/git/pktline"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/streamio"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) InfoRefsUploadPack(in *gitalypb.InfoRefsRequest, stream gitalypb.SmartHTTPService_InfoRefsUploadPackServer) error {
	w := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&gitalypb.InfoRefsResponse{Data: p})
	})
	return handleInfoRefs(stream.Context(), "upload-pack", in, w)
}

func (s *server) InfoRefsReceivePack(in *gitalypb.InfoRefsRequest, stream gitalypb.SmartHTTPService_InfoRefsReceivePackServer) error {
	w := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&gitalypb.InfoRefsResponse{Data: p})
	})
	return handleInfoRefs(stream.Context(), "receive-pack", in, w)
}

func handleInfoRefs(ctx context.Context, service string, req *gitalypb.InfoRefsRequest, w io.Writer) error {
	grpc_logrus.Extract(ctx).WithFields(log.Fields{
		"service": service,
	}).Debug("handleInfoRefs")

	env := git.AddGitProtocolEnv(ctx, req, []string{})

	repoPath, err := helper.GetRepoPath(req.Repository)
	if err != nil {
		return err
	}

	// In case the repository belongs to an object pool, we want to prevent
	// Git from including the pool's refs in the ref advertisement. We do
	// this by rigging core.alternateRefsCommand to produce no output.
	// Because Git itself will append the pool repository directory, the
	// command ends with a "#". The end result is that Git runs
	// `/bin/sh -c 'exit 0 # /path/to/pool.git`.
	args := []string{"-c", "core.alternateRefsCommand=exit 0 #"}
	for _, params := range req.GitConfigOptions {
		args = append(args, "-c", params)
	}

	args = append(args, service, "--stateless-rpc", "--advertise-refs", repoPath)

	cmd, err := git.BareCommand(ctx, nil, nil, nil, env, args...)

	if err != nil {
		if _, ok := status.FromError(err); ok {
			return err
		}
		return status.Errorf(codes.Internal, "GetInfoRefs: cmd: %v", err)
	}

	if _, err := pktline.WriteString(w, fmt.Sprintf("# service=git-%s\n", service)); err != nil {
		return status.Errorf(codes.Internal, "GetInfoRefs: pktLine: %v", err)
	}

	if err := pktline.WriteFlush(w); err != nil {
		return status.Errorf(codes.Internal, "GetInfoRefs: pktFlush: %v", err)
	}

	if _, err := io.Copy(w, cmd); err != nil {
		return status.Errorf(codes.Internal, "GetInfoRefs: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return status.Errorf(codes.Internal, "GetInfoRefs: %v", err)
	}

	return nil
}
