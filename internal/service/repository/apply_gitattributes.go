package repository

import (
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (server) ApplyGitattributes(ctx context.Context, in *pb.ApplyGitattributesRequest) (*pb.ApplyGitattributesResponse, error) {
	repoPath, err := helper.GetRepoPath(in.GetRepository())
	if err != nil {
		return nil, err
	}

	if err := git.ValidateRevision(in.GetRevision()); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "ApplyGitAttributes: revision: %v", err)
	}

	infoPath := path.Join(repoPath, "info")
	attributesPath := path.Join(infoPath, "attributes")
	objectSpec := string(in.GetRevision()) + ":.gitattributes"
	args := []string{"--git-dir", repoPath, "cat-file", "blob", objectSpec}

	cmd, err := helper.GitCommandReader(ctx, args...)
	if err != nil {
		return nil, err
	}
	defer cmd.Close()

	// Create  /info folder if it doesn't exist
	if _, err := os.Stat(infoPath); os.IsNotExist(err) {
		if err := os.Mkdir(infoPath, 0755); err != nil {
			return nil, err
		}
	}

	tempFile, err := ioutil.TempFile(infoPath, "attributes")
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "ApplyGitAttributes: creating temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write attributes to temp file
	if _, err := io.Copy(tempFile, cmd); err != nil {
		return nil, err
	}

	if err := tempFile.Close(); err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		grpc_logrus.Extract(ctx).WithError(err).Info(
			"removing info/attributes because of error on git-show (likely because of missing .gitattributes)")

		err := os.Remove(attributesPath)
		// Ignore error if atttributes file doesn't exist
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		return &pb.ApplyGitattributesResponse{}, nil
	}

	// Rename temp file
	if err := os.Rename(tempFile.Name(), attributesPath); err != nil {
		return nil, err
	}

	return &pb.ApplyGitattributesResponse{}, nil
}
