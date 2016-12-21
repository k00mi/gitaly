package helper

import (
	"net"
	"regexp"

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
	commandCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "daemon_command_count",
			Help: "Count of executed commands, partitioned by subcommand.",
		},
		[]string{"subcommand"},
	)
)

func init() {
	prometheus.MustRegister(connectionsTotal)
	prometheus.MustRegister(requestsBytes)
	prometheus.MustRegister(responseBytes)
	prometheus.MustRegister(commandCount)
}

func ExtractSubcommand(args []string) string {
	for _, subCmd := range args {
		if matches, _ := regexp.MatchString("^[^-][a-z\\-]*$", subCmd); matches {
			return subCmd
		}
	}
	return ""
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

func LogCommand(args []string) {
	commandCount.WithLabelValues(ExtractSubcommand(args)).Inc()
}
