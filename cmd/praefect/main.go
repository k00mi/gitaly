package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitaly/internal/praefect"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
	"gitlab.com/gitlab-org/labkit/tracing"
)

var (
	flagConfig = flag.String("config", "", "Location for the config.toml")
	logger     = logrus.New()

	errNoConfigFile = errors.New("the config flag must be passed")
)

func main() {
	flag.Parse()

	conf, err := configure()
	if err != nil {
		logger.Fatal(err)
	}

	l, err := net.Listen("tcp", conf.ListenAddr)
	if err != nil {
		logger.Fatalf("%s", err)
	}

	logger.WithField("address", conf.ListenAddr).Info("listening at tcp address")
	logger.Fatalf("%v", run(l, conf))
}

func configure() (config.Config, error) {
	var conf config.Config

	if *flagConfig == "" {
		return conf, errNoConfigFile
	}

	conf, err := config.FromFile(*flagConfig)
	if err != nil {
		return conf, fmt.Errorf("error reading config file: %v", err)
	}

	if err := conf.Validate(); err != nil {
		return conf, err
	}

	logger = conf.ConfigureLogger()
	tracing.Initialize(tracing.WithServiceName("praefect"))

	if conf.PrometheusListenAddr != "" {
		logger.WithField("address", conf.PrometheusListenAddr).Info("Starting prometheus listener")
		promMux := http.NewServeMux()
		promMux.Handle("/metrics", promhttp.Handler())

		go func() {
			http.ListenAndServe(conf.PrometheusListenAddr, promMux)
		}()
	}

	return conf, nil
}

func run(l net.Listener, conf config.Config) error {
	srv := praefect.NewServer(nil, logger)

	signals := []os.Signal{syscall.SIGTERM, syscall.SIGINT}
	termCh := make(chan os.Signal, len(signals))
	signal.Notify(termCh, signals...)

	serverErrors := make(chan error, 1)
	go func() { serverErrors <- srv.Start(l) }()

	for _, gitaly := range conf.GitalyServers {
		srv.RegisterNode(gitaly.Name, gitaly.ListenAddr)

		logger.WithField("gitaly listen addr", gitaly.ListenAddr).Info("registered gitaly node")
	}

	var err error
	select {
	case s := <-termCh:
		logger.WithField("signal", s).Warn("received signal, shutting down gracefully")

		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		if shutdownErr := srv.Shutdown(ctx); shutdownErr != nil {
			logger.Warnf("error received during shutting down: %v", shutdownErr)
		}
		err = fmt.Errorf("received signal: %v", s)
	case err = <-serverErrors:
	}

	return err
}
