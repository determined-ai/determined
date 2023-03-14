package dispatcherrm

import (
	prom "github.com/prometheus/client_golang/prometheus"
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
