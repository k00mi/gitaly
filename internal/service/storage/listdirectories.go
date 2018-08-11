package storage

import (
	"os"
	"path"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// This function won't scale, as it walks the dir structure in lexical order,
// which means that it first will
func (s *server) ListDirectories(req *pb.ListDirectoriesRequest, stream pb.StorageService_ListDirectoriesServer) error {
	if helper.ContainsPathTraversal(req.GetPath()) {
		return status.Error(codes.InvalidArgument, "ListDirectories: path traversal is not allowed")
	}

	storageDir, err := helper.GetStorageByName(req.StorageName)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "storage lookup failed: %v", err)
	}

	fullPath := path.Join(storageDir, req.GetPath())
	fi, err := os.Stat(fullPath)
	if os.IsNotExist(err) || !fi.IsDir() {
		return status.Errorf(codes.NotFound, "ListDirectories: %v", err)
	}

	maxDepth := dirDepth(storageDir) + req.GetDepth()
	for _, dir := range directoriesInPath(fullPath) {
		stream.Send(&pb.ListDirectoriesResponse{Paths: recursiveDirs(dir, maxDepth)})
	}

	return nil
}

// Depends on the fact that input is always a path to a dir, not a file
func dirDepth(dir string) uint32 {
	dirs, _ := path.Split(dir)

	return uint32(len(dirs) + 1)
}

func recursiveDirs(dir string, maxDepth uint32) []string {
	var queue, dirs []string

	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]

		subDirs := directoriesInPath(dir)

		if dirDepth(dir)+1 <= maxDepth {
			queue = append(queue, subDirs...)
		}

		dirs = append(dirs, subDirs...)
	}

	return dirs
}

func directoriesInPath(path string) []string {
	fi, err := os.Open(path)
	if os.IsNotExist(err) || err != nil {
		return []string{}
	}

	// Readdirnames returns an empty slice if an error occurs. Given we can't
	// recover, we ignore possible errors here
	dirs, _ := fi.Readdirnames(0)

	return dirs
}
