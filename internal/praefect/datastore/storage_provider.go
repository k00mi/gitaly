package datastore

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/prometheus/client_golang/prometheus"
)

// SecondariesProvider should provide information about secondary storages.
type SecondariesProvider interface {
	// GetConsistentSecondaries returns all secondaries with the same generation as the primary.
	GetConsistentSecondaries(ctx context.Context, virtualStorage, relativePath, primary string) (map[string]struct{}, error)
}

// DirectStorageProvider provides the latest state of the synced nodes.
type DirectStorageProvider struct {
	sp          SecondariesProvider
	errorsTotal *prometheus.CounterVec
}

// NewDirectStorageProvider returns a new storage provider.
func NewDirectStorageProvider(sp SecondariesProvider) *DirectStorageProvider {
	csp := &DirectStorageProvider{
		sp: sp,
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gitaly_praefect_uptodate_storages_errors_total",
				Help: "Total number of errors raised during defining up to date storages for reads distribution",
			},
			[]string{"type"},
		),
	}

	return csp
}

func (c *DirectStorageProvider) Describe(descs chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, descs)
}

func (c *DirectStorageProvider) Collect(collector chan<- prometheus.Metric) {
	c.errorsTotal.Collect(collector)
}

func (c *DirectStorageProvider) GetSyncedNodes(ctx context.Context, virtualStorage, relativePath, primaryStorage string) []string {
	upToDateStorages, err := c.sp.GetConsistentSecondaries(ctx, virtualStorage, relativePath, primaryStorage)
	if err != nil {
		c.errorsTotal.WithLabelValues("retrieve").Inc()
		// this is recoverable error - we can proceed with primary node
		ctxlogrus.Extract(ctx).WithError(err).Warn("get consistent secondaries")
		return []string{primaryStorage}
	}

	storages := make([]string, 0, len(upToDateStorages)+1)
	for upToDateStorage := range upToDateStorages {
		if upToDateStorage != primaryStorage {
			storages = append(storages, upToDateStorage)
		}
	}

	return append(storages, primaryStorage)
}
