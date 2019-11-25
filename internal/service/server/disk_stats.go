package server

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"gitlab.com/gitlab-org/gitaly/internal/config"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"golang.org/x/sys/unix"
)

func (s *server) DiskStatistics(ctx context.Context, _ *gitalypb.DiskStatisticsRequest) (*gitalypb.DiskStatisticsResponse, error) {
	var results []*gitalypb.DiskStatisticsResponse_StorageStatus
	for _, shard := range config.Config.Storages {
		shardInfo, err := getStorageStatus(shard)
		if err != nil {
			ctxlogrus.Extract(ctx).WithField("storage", shard).WithError(err).Error("to retrieve shard disk statistics")
			results = append(results, &gitalypb.DiskStatisticsResponse_StorageStatus{StorageName: shard.Name})
			continue
		}

		results = append(results, shardInfo)
	}

	return &gitalypb.DiskStatisticsResponse{
		StorageStatuses: results,
	}, nil
}

func getStorageStatus(shard config.Storage) (*gitalypb.DiskStatisticsResponse_StorageStatus, error) {
	var stats unix.Statfs_t
	err := unix.Statfs(shard.Path, &stats)
	if err != nil {
		return nil, err
	}

	// Redundant conversions to handle differences between unix families
	available := int64(stats.Bavail) * int64(stats.Bsize)
	used := (int64(stats.Blocks) - int64(stats.Bfree)) * int64(stats.Bsize)

	return &gitalypb.DiskStatisticsResponse_StorageStatus{
		StorageName: shard.Name,
		Available:   available,
		Used:        used,
	}, nil
}
