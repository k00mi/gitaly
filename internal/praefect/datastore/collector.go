package datastore

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var descReadOnlyRepositories = prometheus.NewDesc(
	"gitaly_praefect_read_only_repositories",
	"Number of repositories in read-only mode within a virtual storage.",
	[]string{"virtual_storage"},
	nil,
)

// PrimaryGetter is an interface used by RepositoryStoreCollector to
// get information of primary assignments of the configured virtual
// storages.
type PrimaryGetter interface {
	// GetPrimaries returns primaries by their virtual storages. If a virtual
	// storage does not have a primary, its entry should be an empty string.
	GetPrimaries(ctx context.Context) (map[string]string, error)
}

// RepositoryStoreCollector collects metrics from the RepositoryStore.
type RepositoryStoreCollector struct {
	log logrus.FieldLogger
	rs  RepositoryStore
	pg  PrimaryGetter
}

// NewRepositoryStoreCollector returns a new collector.
func NewRepositoryStoreCollector(log logrus.FieldLogger, rs RepositoryStore, pg PrimaryGetter) *RepositoryStoreCollector {
	return &RepositoryStoreCollector{log, rs, pg}
}

func (c *RepositoryStoreCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *RepositoryStoreCollector) Collect(ch chan<- prometheus.Metric) {
	vsPrimaries, err := c.pg.GetPrimaries(context.TODO())
	if err != nil {
		c.log.WithError(err).Error("RepositoryStoreCollector: get virtual storage primaries")
		return
	}

	readOnlyCounts, err := c.rs.CountReadOnlyRepositories(context.TODO(), vsPrimaries)
	if err != nil {
		c.log.WithError(err).Error("RepositoryStoreCollector: count read-only repositories")
		return
	}

	for virtualStorage, readOnlyCount := range readOnlyCounts {
		ch <- prometheus.MustNewConstMetric(descReadOnlyRepositories, prometheus.GaugeValue, float64(readOnlyCount), virtualStorage)
	}
}
