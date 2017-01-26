package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"gitlab.com/gitlab-org/gitaly/internal/router"
)

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

	serverError := make(chan error, 2)
	go func() {
		serverError <- http.Serve(listener, router.NewRouter())
	}()

	if config.PrometheusListenAddr != "" {
		promMux := http.NewServeMux()
		promMux.Handle("/metrics", promhttp.Handler())
		go func() {
			http.ListenAndServe(config.PrometheusListenAddr, promMux)
		}()
	}

	log.Fatal(<-serverError)
}
