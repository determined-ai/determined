package dispatcherrm

import (
	"time"

	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/determined-ai/determined/master/internal/config"
)

const (
	promNamespace = "determined"
	promSubsystem = "dispatcherrm"
)

var (
	dispatcherLabels    = []string{"method"}
	dispatcherHistogram = prom.NewHistogramVec(prom.HistogramOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "seconds",
		Help:      "duration of dispatcher API calls",
		Buckets:   prom.DefBuckets,
	}, dispatcherLabels)
	dispatcherErrors = prom.NewCounterVec(prom.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "errors",
		Help:      "errors from dispatcher API calls",
	}, dispatcherLabels)
)

func init() {
	prom.MustRegister(dispatcherHistogram)
	prom.MustRegister(dispatcherErrors)
}

func recordAPITiming(labels ...string) (end func()) {
	if !config.GetMasterConfig().Observability.EnablePrometheus {
		return func() {}
	}

	start := time.Now()
	return func() {
		dispatcherHistogram.WithLabelValues(labels...).Observe(time.Since(start).Seconds())
	}
}

func recordAPIErr(labels ...string) func(error) {
	if !config.GetMasterConfig().Observability.EnablePrometheus {
		return func(error) {}
	}

	return func(err error) {
		if err == nil {
			return
		}
		dispatcherErrors.WithLabelValues(labels...).Inc()
	}
}
