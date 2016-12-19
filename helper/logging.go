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
	requestsBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "daemon_requests_bytes",
			Help: "Bytesize of incoming requests by gitaly, partitioned by source.",
		},
		[]string{"source"},
	)
	responseBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "daemon_response_bytes",
			Help: "Bytesize of outcoming responses by gitaly, partitioned by source.",
		},
		[]string{"source"},
	)
)

func init() {
	prometheus.MustRegister(connectionsTotal)
	prometheus.MustRegister(requestsBytes)
	prometheus.MustRegister(responseBytes)
}

func LogConnection(conn net.Conn) {
	protocol := ""

	switch conn.(type) {
	case *net.TCPConn:
		protocol = "TCP"
	}

	connectionsTotal.WithLabelValues(protocol).Inc()
}

func LogMessage(msg []byte) {
	bytes := float64(len(msg))

	requestsBytes.WithLabelValues("gitaly").Add(bytes)
}

func LogResponse(msg []byte) {
	bytes := float64(len(msg))

	responseBytes.WithLabelValues("gitaly").Add(bytes)
}
