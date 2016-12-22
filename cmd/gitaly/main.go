package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	serv "gitlab.com/gitlab-org/gitaly/server"
)

var (
	prometheusListenAddr = flag.String("prometheusListenAddr", "", "Prometheus listening address, e.g. ':9100'")
)

func main() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	server := serv.NewServer()
	go func() {
		server.Serve("0.0.0.0:6666", serv.CommandExecutor)
	}()

	if *prometheusListenAddr != "" {
		promMux := http.NewServeMux()
		promMux.Handle("/metrics", promhttp.Handler())
		go func() {
			http.ListenAndServe(*prometheusListenAddr, promMux)
		}()
	}

	select {
	case <-ch:
		server.Stop()
	}
}
