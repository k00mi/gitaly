package repository

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	log "github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitaly/internal/helper/housekeeping"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (server) GarbageCollect(ctx context.Context, in *pb.GarbageCollectRequest) (*pb.GarbageCollectResponse, error) {
	ctxlogger := grpc_logrus.Extract(ctx)
	ctxlogger.WithFields(log.Fields{
		"WriteBitmaps": in.GetCreateBitmap(),
	}).Debug("GarbageCollect")

	repoPath, err := helper.GetRepoPath(in.GetRepository())
	if err != nil {
		return nil, err
	}

	threshold := time.Now().Add(-1 * time.Hour)
	if err := cleanRefsLocks(filepath.Join(repoPath, "refs"), threshold); err != nil {
		return nil, status.Errorf(codes.Internal, "GarbageCollect: cleanRefsLocks: %v", err)
	}
	if err := cleanPackedRefsLock(repoPath, threshold); err != nil {
		return nil, status.Errorf(codes.Internal, "GarbageCollect: cleanPackedRefsLock: %v", err)
	}

	args := []string{"-c"}
	if in.GetCreateBitmap() {
		args = append(args, "repack.writeBitmaps=true")
	} else {
		args = append(args, "repack.writeBitmaps=false")
	}
	args = append(args, "gc")
	cmd, err := git.Command(ctx, in.GetRepository(), args...)
	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "GarbageCollect: gitCommand: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, status.Errorf(codes.Internal, "GarbageCollect: cmd wait: %v", err)
	}

	// Perform housekeeping post GC
	err = housekeeping.Perform(ctx, repoPath)
	if err != nil {
		ctxlogger.WithError(err).Warn("Post gc housekeeping failed")
	}

	return &pb.GarbageCollectResponse{}, nil
}

func cleanRefsLocks(rootPath string, threshold time.Time) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(info.Name(), ".lock") && info.ModTime().Before(threshold) {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}

		return nil
	})
}

func cleanPackedRefsLock(repoPath string, threshold time.Time) error {
	path := filepath.Join(repoPath, "packed-refs.lock")
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if fileInfo.ModTime().Before(threshold) {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}
