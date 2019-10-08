package git

import (
	"context"
	"os"
	"strconv"
	"strings"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/gitaly/internal/git/packfile"
)

var badBitmapRequestCount = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "gitaly_bad_bitmap_request_total",
		Help: "RPC calls during which there was not exactly 1 packfile bitmap",
	},
	[]string{"method", "bitmaps"},
)

func init() { prometheus.MustRegister(badBitmapRequestCount) }

// WarnIfTooManyBitmaps checks for too many (more than one) bitmaps in
// repoPath, and if it finds any, it logs a warning. This is to help us
// investigate https://gitlab.com/gitlab-org/gitaly/issues/1728.
func WarnIfTooManyBitmaps(ctx context.Context, repoPath string) {
	logEntry := grpc_logrus.Extract(ctx)

	objdirs, err := ObjectDirectories(ctx, repoPath)
	if err != nil {
		logEntry.WithError(err).Info("bitmap check failed")
		return
	}

	var count int
	seen := make(map[string]bool)
	for _, dir := range objdirs {
		if seen[dir] {
			continue
		}
		seen[dir] = true

		packs, err := packfile.List(dir)
		if err != nil {
			logEntry.WithError(err).Info("bitmap check failed")
			return
		}

		for _, p := range packs {
			fi, err := os.Stat(strings.TrimSuffix(p, ".pack") + ".bitmap")
			if err == nil && !fi.IsDir() {
				count++
			}
		}
	}

	if count == 1 {
		// Exactly one bitmap: this is how things should be.
		return
	}

	if count > 1 {
		logEntry.WithField("bitmaps", count).Warn("found more than one packfile bitmap in repository")
	}

	// The case where count == 0 is likely to occur early in the life of a
	// repository. We don't want to spam our logs with that, so we count but
	// not log it.

	grpcMethod, ok := grpc_ctxtags.Extract(ctx).Values()["grpc.request.fullMethod"].(string)
	if !ok {
		return
	}

	badBitmapRequestCount.WithLabelValues(grpcMethod, strconv.Itoa(count)).Inc()
}