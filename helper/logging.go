package helper

import (
	"net"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	connectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "daemon_connections_total",
			Help: "How many connections have been taken by gitaly, partitioned by protocol.",
		},
		[]string{"protocol"},
	)
)

func init() {
	prometheus.MustRegister(connectionsTotal)
}

func LogConnection(conn net.Conn) {
	protocol := ""

	switch conn.(type) {
	case *net.TCPConn:
		protocol = "TCP"
	}

	connectionsTotal.WithLabelValues(protocol).Inc()
}
