package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitaly/internal/config"
	"gitlab.com/gitlab-org/gitaly/internal/connectioncounter"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/linguist"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
	"gitlab.com/gitlab-org/gitaly/internal/server"
	"gitlab.com/gitlab-org/gitaly/internal/tempdir"
	"gitlab.com/gitlab-org/gitaly/internal/version"
)

var (
	flagVersion = flag.Bool("version", false, "Print version and exit")
)

func loadConfig(configPath string) error {
	cfgFile, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer cfgFile.Close()

	if err = config.Load(cfgFile); err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	if err := linguist.LoadColors(); err != nil {
		return fmt.Errorf("load linguist colors: %v", err)
	}

	return nil
}

// registerServerVersionPromGauge registers a label with the current server version
// making it easy to see what versions of Gitaly are running across a cluster
func registerServerVersionPromGauge() {
	gitVersion, err := git.Version()
	if err != nil {
		fmt.Printf("git version: %v\n", err)
		os.Exit(1)
	}
	gitlabBuildInfoGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_build_info",
		Help: "Current build info for this GitLab Service",
		ConstLabels: prometheus.Labels{
			"version":     version.GetVersion(),
			"built":       version.GetBuildTime(),
			"git_version": gitVersion,
		},
	})

	prometheus.MustRegister(gitlabBuildInfoGauge)
	gitlabBuildInfoGauge.Set(1)
}

func flagUsage() {
	fmt.Println(version.GetVersionString())
	fmt.Printf("Usage: %v [OPTIONS] configfile\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = flagUsage
	flag.Parse()

	// If invoked with -version
	if *flagVersion {
		fmt.Println(version.GetVersionString())
		os.Exit(0)
	}

	if flag.NArg() != 1 || flag.Arg(0) == "" {
		flag.Usage()
		os.Exit(2)
	}

	log.WithField("version", version.GetVersionString()).Info("Starting Gitaly")
	registerServerVersionPromGauge()

	configPath := flag.Arg(0)
	if err := loadConfig(configPath); err != nil {
		log.WithError(err).WithField("config_path", configPath).Fatal("load config")
	}

	config.ConfigureLogging()
	config.ConfigureSentry(version.GetVersion())
	config.ConfigurePrometheus()
	config.ConfigureConcurrencyLimits()

	tempdir.StartCleaning()

	var insecureListeners []net.Listener
	var secureListeners []net.Listener

	if socketPath := config.Config.SocketPath; socketPath != "" {
		l, err := createUnixListener(socketPath)
		if err != nil {
			log.WithError(err).Fatal("configure unix listener")
		}
		log.WithField("address", socketPath).Info("listening on unix socket")
		insecureListeners = append(insecureListeners, l)
	}

	if addr := config.Config.ListenAddr; addr != "" {
		l, err := net.Listen("tcp", addr)
		if err != nil {
			log.WithError(err).Fatal("configure tcp listener")
		}

		log.WithField("address", addr).Info("listening at tcp address")
		insecureListeners = append(insecureListeners, connectioncounter.New("tcp", l))
	}

	if addr := config.Config.TLSListenAddr; addr != "" {
		tlsListener, err := net.Listen("tcp", addr)
		if err != nil {
			log.WithError(err).Fatal("configure tls listener")
		}

		secureListeners = append(secureListeners, connectioncounter.New("tls", tlsListener))
	}

	if config.Config.PrometheusListenAddr != "" {
		log.WithField("address", config.Config.PrometheusListenAddr).Info("Starting prometheus listener")
		promMux := http.NewServeMux()
		promMux.Handle("/metrics", promhttp.Handler())

		server.AddPprofHandlers(promMux)

		go func() {
			http.ListenAndServe(config.Config.PrometheusListenAddr, promMux)
		}()
	}

	log.WithError(run(insecureListeners, secureListeners)).Fatal("shutting down")
}

func createUnixListener(socketPath string) (net.Listener, error) {
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	l, err := net.Listen("unix", socketPath)
	return connectioncounter.New("unix", l), err
}

// Inside here we can use deferred functions. This is needed because
// log.Fatal bypasses deferred functions.
func run(insecureListeners, secureListeners []net.Listener) error {
	signals := []os.Signal{syscall.SIGTERM, syscall.SIGINT}
	termCh := make(chan os.Signal, len(signals))
	signal.Notify(termCh, signals...)

	ruby, err := rubyserver.Start()
	if err != nil {
		return err
	}
	defer ruby.Stop()

	serverErrors := make(chan error, len(insecureListeners)+len(secureListeners))
	if len(insecureListeners) > 0 {
		insecureServer := server.NewInsecure(ruby)
		defer insecureServer.Stop()

		for _, listener := range insecureListeners {
			// Must pass the listener as a function argument because there is a race
			// between 'go' and 'for'.
			go func(l net.Listener) {
				serverErrors <- insecureServer.Serve(l)
			}(listener)
		}
	}

	if len(secureListeners) > 0 {
		secureServer := server.NewSecure(ruby)
		defer secureServer.Stop()

		for _, listener := range secureListeners {
			go func(l net.Listener) {
				serverErrors <- secureServer.Serve(l)
			}(listener)
		}
	}

	select {
	case s := <-termCh:
		err = fmt.Errorf("received signal %q", s)
	case err = <-serverErrors:
	}

	return err
}
