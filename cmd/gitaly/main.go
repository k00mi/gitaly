package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	serv "gitlab.com/gitlab-org/gitaly/server"
)

type Config struct {
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

	go func() {
		server.Serve("0.0.0.0:6666", serv.CommandExecutor)
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
		server.Stop()
	}
}
