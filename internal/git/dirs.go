package git

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
)

// alternateOutsideStorageError is returned when an alternates file contains an
// alternate which is not within the storage's root.
type alternateOutsideStorageError string

func (path alternateOutsideStorageError) Error() string {
	return fmt.Sprintf("alternate %q is outside the storage root", string(path))
}

// ObjectDirectories looks for Git object directories, including
// alternates specified in objects/info/alternates.
//
// CAVEAT Git supports quoted strings in here, but we do not. We should
// never need those on a Gitaly server.
func ObjectDirectories(ctx context.Context, storageRoot, repoPath string) ([]string, error) {
	objDir := filepath.Join(repoPath, "objects")
	return altObjectDirs(ctx, storageRoot+string(os.PathSeparator), objDir, 0)
}

func altObjectDirs(ctx context.Context, storagePrefix, objDir string, depth int) ([]string, error) {
	logEntry := grpc_logrus.Extract(ctx)

	const maxAlternatesDepth = 5 // Taken from https://github.com/git/git/blob/v2.23.0/sha1-file.c#L575
	if depth > maxAlternatesDepth {
		logEntry.WithField("objdir", objDir).Warn("ignoring deeply nested alternate object directory")
		return nil, nil
	}

	fi, err := os.Stat(objDir)
	if os.IsNotExist(err) {
		logEntry.WithField("objdir", objDir).Warn("object directory not found")
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, nil
	}

	dirs := []string{objDir}

	alternates, err := ioutil.ReadFile(filepath.Join(objDir, "info", "alternates"))
	if os.IsNotExist(err) {
		return dirs, nil
	}
	if err != nil {
		return nil, err
	}

	for _, newDir := range strings.Split(string(alternates), "\n") {
		if len(newDir) == 0 || newDir[0] == '#' {
			continue
		}

		if !filepath.IsAbs(newDir) {
			newDir = filepath.Join(objDir, newDir)
		}

		if !strings.HasPrefix(newDir, storagePrefix) {
			return nil, alternateOutsideStorageError(newDir)
		}

		nestedDirs, err := altObjectDirs(ctx, storagePrefix, newDir, depth+1)
		if err != nil {
			return nil, err
		}

		dirs = append(dirs, nestedDirs...)
	}

	return dirs, nil
}
