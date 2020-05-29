package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace           = "gitaly"
	subsystem           = "praefect"
	labelVirtualStorage = "virtual_storage"
	labelGitalyStorage  = "gitaly_storage"
)

// StorageGauge is a metric wrapper that abstracts and simplifies the interface
// of the underlying type. It is intended for gauges that are scoped by virtual
// storage and by Gitaly storage.
type StorageGauge struct {
	gv *prometheus.GaugeVec
}

func newStorageGauge(name string) StorageGauge {
	sg := StorageGauge{}
	sg.gv = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
		},
		[]string{labelVirtualStorage, labelGitalyStorage},
	)
	return sg
}

// Inc will inc the gauge for the specified virtual and gitaly storage
func (sg StorageGauge) Inc(virtualStorage, gitalyStorage string) {
	sg.gv.WithLabelValues(virtualStorage, gitalyStorage).Inc()
}

// Dec will dec the gauge for the specified virtual and gitaly storage
func (sg StorageGauge) Dec(virtualStorage, gitalyStorage string) {
	sg.gv.WithLabelValues(virtualStorage, gitalyStorage).Dec()
}
