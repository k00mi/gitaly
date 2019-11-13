package server

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"golang.org/x/sync/errgroup"
)

// DiskStatistics sends DiskStatisticsRequest to all of a praefect server's internal gitaly nodes and aggregates the
// results into a response
func (s *Server) DiskStatistics(ctx context.Context, _ *gitalypb.DiskStatisticsRequest) (*gitalypb.DiskStatisticsResponse, error) {
	storageStatuses := make([][]*gitalypb.DiskStatisticsResponse_StorageStatus, len(s.conf.Nodes))

	g, ctx := errgroup.WithContext(ctx)

	for i, node := range s.conf.Nodes {
		i := i // necessary since it will be used in a goroutine below
		node := node
		cc, err := s.clientCC.GetConnection(node.Storage)
		if err != nil {
			return nil, helper.ErrInternalf("error getting client connection for %s: %v", node.Storage, err)
		}

		g.Go(func() error {
			client := gitalypb.NewServerServiceClient(cc)
			resp, err := client.DiskStatistics(ctx, &gitalypb.DiskStatisticsRequest{})
			if err != nil {
				return fmt.Errorf("error when requesting disk statistics from internal storage %v", node.Storage)
			}

			storageStatuses[i] = resp.GetStorageStatuses()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, helper.ErrInternal(err)
	}

	var response gitalypb.DiskStatisticsResponse

	for _, storageStatus := range storageStatuses {
		response.StorageStatuses = append(response.StorageStatuses, storageStatus...)
	}

	return &response, nil
}
