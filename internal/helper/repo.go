package helper

import (
	"gitlab.com/gitlab-org/gitaly/internal/git/repository"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RepoPathEqual compares if two repositories are in the same location
func RepoPathEqual(a, b repository.GitRepo) bool {
	return a.GetStorageName() == b.GetStorageName() &&
		a.GetRelativePath() == b.GetRelativePath()
}

// GetStorageByName will return the path for the storage, which is fetched by
// its key. An error is return if it cannot be found.
// Deprecated: please use storage.Locator to define the storage path.
func GetStorageByName(storageName string) (string, error) {
	storagePath, ok := config.Config.StoragePath(storageName)
	if !ok {
		return "", status.Errorf(codes.InvalidArgument, "Storage can not be found by name '%s'", storageName)
	}

	return storagePath, nil
}

// ProtoRepoFromRepo allows object pools and repository abstractions to be used
// in places that require a concrete type
func ProtoRepoFromRepo(repo repository.GitRepo) *gitalypb.Repository {
	return &gitalypb.Repository{
		StorageName:                   repo.GetStorageName(),
		GitAlternateObjectDirectories: repo.GetGitAlternateObjectDirectories(),
		GitObjectDirectory:            repo.GetGitObjectDirectory(),
		RelativePath:                  repo.GetRelativePath(),
	}
}
