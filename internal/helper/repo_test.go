package helper

import (
	"testing"

	"gitlab.com/gitlab-org/gitaly/internal/config"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"github.com/stretchr/testify/assert"
)

func TestGetRepoPath(t *testing.T) {
	defer func(oldStorages []config.Storage) {
		config.Config.Storages = oldStorages
	}(config.Config.Storages)

	exampleStorages := []config.Storage{
		{Name: "default", Path: "/home/git/repositories1"},
		{Name: "other", Path: "/home/git/repositories2"},
		{Name: "third", Path: "/home/git/repositories3"},
	}

	testCases := []struct {
		desc     string
		storages []config.Storage
		repo     *pb.Repository
		path     string
		notFound bool
	}{
		{
			desc:     "storages configured but only repo.Path is provided",
			storages: exampleStorages,
			repo:     &pb.Repository{Path: "/foo/bar.git"},
			path:     "/foo/bar.git",
		},
		{
			desc:     "storages configured, storage name not known, repo.Path provided",
			storages: exampleStorages,
			repo:     &pb.Repository{Path: "/foo/bar.git", StorageName: "does not exist", RelativePath: "foobar.git"},
			path:     "/foo/bar.git",
		},
		{
			desc: "no storages configured, repo.Path provided",
			repo: &pb.Repository{Path: "/foo/bar.git", StorageName: "does not exist", RelativePath: "foobar.git"},
			path: "/foo/bar.git",
		},
		{
			desc:     "storages configured, no repo.Path",
			storages: exampleStorages,
			repo:     &pb.Repository{StorageName: "default", RelativePath: "bazqux.git"},
			path:     "/home/git/repositories1/bazqux.git",
		},
		{
			desc:     "storage configured, storage name match, repo.Path provided",
			storages: exampleStorages,
			repo:     &pb.Repository{Path: "/foo/bar.git", StorageName: "default", RelativePath: "bazqux.git"},
			path:     "/home/git/repositories1/bazqux.git",
		},
		{
			desc: "no storage config, repo.Path provided",
			repo: &pb.Repository{Path: "/foo/bar.git", StorageName: "default", RelativePath: "bazqux.git"},
			path: "/foo/bar.git",
		},
		{
			desc:     "no storage config, storage name provided, no repo.Path",
			repo:     &pb.Repository{StorageName: "does not exist", RelativePath: "foobar.git"},
			notFound: true,
		},
		{
			desc:     "no storage config, nil repo",
			notFound: true,
		},
		{
			desc:     "storage config provided, empty repo",
			storages: exampleStorages,
			repo:     &pb.Repository{},
			notFound: true,
		},
		{
			desc:     "no storage config, empty repo",
			repo:     &pb.Repository{},
			notFound: true,
		},
	}

	for _, tc := range testCases {
		config.Config.Storages = tc.storages
		path, err := GetRepoPath(tc.repo)

		if tc.notFound {
			assert.Error(t, err, tc.desc)
			continue
		}

		if err != nil {
			assert.NoError(t, err, tc.desc)
			continue
		}

		assert.Equal(t, tc.path, path, tc.desc)
	}
}
