package helper

import (
	"os"
	"path"

	"gitlab.com/gitlab-org/gitaly/internal/config"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// GetRepoPath returns the full path of the repository referenced by an RPC
// Repository message. The errors returned are gRPC errors with relevant
// error codes, and need no further decoration.
func GetRepoPath(repo *pb.Repository) (string, error) {
	var repoPath string

	if storagePath, ok := config.StoragePath(repo.GetStorageName()); ok {
		repoPath = path.Join(storagePath, repo.GetRelativePath())
	} else {
		repoPath = repo.GetPath()
	}

	if repoPath == "" {
		return "", grpc.Errorf(codes.InvalidArgument, "GetRepoPath: empty repo")
	}

	if _, err := os.Stat(path.Join(repoPath, "objects")); err != nil {
		return "", grpc.Errorf(codes.NotFound, "GetRepoPath: not a git repository '%s'", repoPath)
	}

	return repoPath, nil
}
