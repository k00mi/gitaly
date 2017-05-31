package config

import (
	"github.com/getsentry/raven-go"
	log "github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitaly/internal/service/middleware/panichandler"
)

// ConfigureSentry configures the sentry DSN
func ConfigureSentry(version string) {
	if Config.Logging.SentryDSN != "" {
		log.Debug("Using sentry logging")
		raven.SetDSN(Config.Logging.SentryDSN)

		panichandler.InstallPanicHandler(func(grpcMethod string, _err interface{}) {
			if err, ok := _err.(error); ok {
				raven.CaptureError(err, map[string]string{
					"grpcMethod": grpcMethod,
					"panic":      "1",
				}, nil)
			}
		})
	}

	if version != "" {
		raven.SetRelease(version)
	}
}
