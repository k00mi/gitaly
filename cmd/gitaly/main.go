package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"gitlab.com/gitlab-org/gitaly/internal/service"
	"gitlab.com/gitlab-org/gitaly/internal/service/middleware/panichandler"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/kelseyhightower/envconfig"
	"github.com/mwitkow/go-grpc-middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Config specifies the gitaly server configuration
type Config struct {
	SocketPath           string `split_words:"true"`
	ListenAddr           string `split_words:"true"`
	PrometheusListenAddr string `split_words:"true"`
}

var version string

func main() {
	log.Println("Starting Gitaly", version)

	config := Config{}
	err := envconfig.Process("gitaly", &config)
	if err != nil {
		log.Fatal(err)
	}

	if config.SocketPath == "" && config.ListenAddr == "" {
		log.Fatal("Must set $GITALY_SOCKET_PATH or $GITALY_LISTEN_ADDR")
	}

	var listeners []net.Listener

	if socketPath := config.SocketPath; socketPath != "" {
		l, err := createUnixListener(socketPath)
		if err != nil {
			log.Fatalf("configure unix listener: %v", err)
		}
		log.Printf("listening on unix socket %q", socketPath)
		listeners = append(listeners, l)
	}

	if addr := config.ListenAddr; addr != "" {
		l, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("configure tcp listener: %v", err)
		}
		log.Printf("listening at tcp address %q", addr)
		listeners = append(listeners, l)
	}

	server := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			panichandler.StreamPanicHandler,         // Panic Handler first: handle panics gracefully
			grpc_prometheus.StreamServerInterceptor, // Prometheus Metrics next: measure RPC times
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			panichandler.UnaryPanicHandler,         // Panic Handler first: handle panics gracefully
			grpc_prometheus.UnaryServerInterceptor, // Prometheus Metrics next: measure RPC times
		)),
	)

	service.RegisterAll(server)
	reflection.Register(server)

	// After all your registrations, make sure all of the Prometheus metrics are initialized.
	grpc_prometheus.Register(server)

	serverError := make(chan error, len(listeners))
	for _, listener := range listeners {
		// Must pass the listener as a function argument because there is a race
		// between 'go' and 'for'.
		go func(l net.Listener) {
			serverError <- server.Serve(l)
		}(listener)
	}

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

func createUnixListener(socketPath string) (net.Listener, error) {
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return net.Listen("unix", socketPath)
}
