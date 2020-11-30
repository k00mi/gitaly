package info

import (
	"context"
	"fmt"
	"sort"

	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func (s *Server) DatalossCheck(ctx context.Context, req *gitalypb.DatalossCheckRequest) (*gitalypb.DatalossCheckResponse, error) {
	configuredStorages, ok := s.conf.StorageNames()[req.VirtualStorage]
	if !ok {
		return nil, fmt.Errorf("unknown virtual storage: %q", req.VirtualStorage)
	}

	shard, err := s.nodeMgr.GetShard(ctx, req.GetVirtualStorage())
	if err != nil {
		return nil, err
	}

	outdatedRepos, err := s.rs.GetOutdatedRepositories(ctx, req.GetVirtualStorage())
	if err != nil {
		return nil, err
	}

	pbRepos := make([]*gitalypb.DatalossCheckResponse_Repository, 0, len(outdatedRepos))
	for relativePath, outdatedStorages := range outdatedRepos {
		readOnly := false

		storages := make(map[string]*gitalypb.DatalossCheckResponse_Repository_Storage, len(configuredStorages))
		for _, storage := range configuredStorages {
			storages[storage] = &gitalypb.DatalossCheckResponse_Repository_Storage{
				Name:     storage,
				Assigned: true,
			}
		}

		for name, behindBy := range outdatedStorages {
			if name == shard.Primary.GetStorage() {
				readOnly = true
			}

			storages[name].BehindBy = int64(behindBy)
		}

		if !req.IncludePartiallyReplicated && !readOnly {
			continue
		}

		storagesSlice := make([]*gitalypb.DatalossCheckResponse_Repository_Storage, 0, len(storages))
		for _, storage := range storages {
			storagesSlice = append(storagesSlice, storage)
		}

		sort.Slice(storagesSlice, func(i, j int) bool { return storagesSlice[i].Name < storagesSlice[j].Name })

		pbRepos = append(pbRepos, &gitalypb.DatalossCheckResponse_Repository{
			RelativePath: relativePath,
			ReadOnly:     readOnly,
			Storages:     storagesSlice,
			Primary:      shard.Primary.GetStorage(),
		})
	}

	sort.Slice(pbRepos, func(i, j int) bool { return pbRepos[i].RelativePath < pbRepos[j].RelativePath })

	return &gitalypb.DatalossCheckResponse{
		Repositories: pbRepos,
	}, nil
}
