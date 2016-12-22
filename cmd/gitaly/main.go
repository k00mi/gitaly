package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	SocketPath           string `split_words:"true"`
	PrometheusListenAddr string `split_words:"true"`
}

func main() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

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

	go func() {
		// TODO: Add a handler
		http.Serve(listener, nil)
	}()

	if config.PrometheusListenAddr != "" {
		promMux := http.NewServeMux()
		promMux.Handle("/metrics", promhttp.Handler())
		go func() {
			http.ListenAndServe(config.PrometheusListenAddr, promMux)
		}()
	}

	select {
	case <-ch:
		log.Println("Received shutdown message")
		listener.Close()
	}
}
