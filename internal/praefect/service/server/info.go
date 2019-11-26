package server

import (
	"context"
	"fmt"
	"sync"

	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"golang.org/x/sync/errgroup"

	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/models"
)

// ServerInfo sends ServerInfoRequest to all of a praefect server's internal gitaly nodes and aggregates the results into
// a response
func (s *Server) ServerInfo(ctx context.Context, in *gitalypb.ServerInfoRequest) (*gitalypb.ServerInfoResponse, error) {
	var once sync.Once
	nodesChecked := make(map[string]struct{})

	var nodes []*models.Node
	for _, virtualStorage := range s.conf.VirtualStorages {
		for _, node := range virtualStorage.Nodes {
			if _, ok := nodesChecked[node.Storage]; ok {
				continue
			}

			nodesChecked[node.Storage] = struct{}{}
			nodes = append(nodes, node)
		}
	}

	var gitVersion, serverVersion string

	g, ctx := errgroup.WithContext(ctx)

	storageStatuses := make([][]*gitalypb.ServerInfoResponse_StorageStatus, len(nodes))

	for i, node := range nodes {
		i := i
		node := node
		cc, err := s.clientCC.GetConnection(node.Storage)
		if err != nil {
			return nil, helper.ErrInternalf("error getting client connection for %s: %v", node.Storage, err)
		}
		g.Go(func() error {
			client := gitalypb.NewServerServiceClient(cc)
			resp, err := client.ServerInfo(ctx, &gitalypb.ServerInfoRequest{})
			if err != nil {
				return fmt.Errorf("error when requesting server info from internal storage %v", node.Storage)
			}

			storageStatuses[i] = resp.GetStorageStatuses()

			if node.DefaultPrimary {
				once.Do(func() {
					gitVersion, serverVersion = resp.GetGitVersion(), resp.GetServerVersion()
				})
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, helper.ErrInternal(err)
	}

	var response gitalypb.ServerInfoResponse

	for _, storageStatus := range storageStatuses {
		response.StorageStatuses = append(response.StorageStatuses, storageStatus...)
	}

	response.GitVersion, response.ServerVersion = gitVersion, serverVersion

	return &response, nil
}
