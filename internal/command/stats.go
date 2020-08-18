package command

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

type requestStatsKey struct{}

type Stats struct {
	registry map[string]int
	sync.Mutex
}

func (stats *Stats) RecordSum(key string, value int) {
	stats.Lock()
	defer stats.Unlock()

	if prevValue, ok := stats.registry[key]; ok {
		value += prevValue
	}

	stats.registry[key] = value
}

func (stats *Stats) RecordMax(key string, value int) {
	stats.Lock()
	defer stats.Unlock()

	if prevValue, ok := stats.registry[key]; ok {
		if prevValue > value {
			return
		}
	}

	stats.registry[key] = value
}

func (stats *Stats) Fields() logrus.Fields {
	stats.Lock()
	defer stats.Unlock()

	f := logrus.Fields{}
	for k, v := range stats.registry {
		f[k] = v
	}
	return f
}

func StatsFromContext(ctx context.Context) *Stats {
	stats, _ := ctx.Value(requestStatsKey{}).(*Stats)
	return stats
}

func InitContextStats(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestStatsKey{}, &Stats{
		registry: make(map[string]int),
	})
}
