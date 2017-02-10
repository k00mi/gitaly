package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"gitlab.com/gitlab-org/gitaly/internal/service"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Config specifies the gitaly server configuration
type Config struct {
	SocketPath           string `split_words:"true"`
	PrometheusListenAddr string `split_words:"true"`
}

func main() {
	config := Config{}
	err := envconfig.Process("gitaly", &config)
	if err != nil {
		log.Fatal(err)
	}

	if config.SocketPath == "" {
		log.Fatal("GITALY_SOCKET_PATH environment variable is not set")
	}

	if err := os.Remove(config.SocketPath); err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	listener, err := net.Listen("unix", config.SocketPath)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening on socket", config.SocketPath)

	server := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)

	service.RegisterAll(server)
	reflection.Register(server)

	// After all your registrations, make sure all of the Prometheus metrics are initialized.
	grpc_prometheus.Register(server)

	serverError := make(chan error, 2)
	go func() {
		serverError <- server.Serve(listener)
	}()

	if config.PrometheusListenAddr != "" {
		log.Print("Starting prometheus listener ", config.PrometheusListenAddr)
		promMux := http.NewServeMux()
		promMux.Handle("/metrics", promhttp.Handler())
		go func() {
			http.ListenAndServe(config.PrometheusListenAddr, promMux)
		}()
	}

	log.Fatal(<-serverError)
}
