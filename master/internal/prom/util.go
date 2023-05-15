package prom

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/determined-ai/determined/master/internal/config"
)

// DeterminedNamespace is the prometheus namespace for Determined metrics.
const DeterminedNamespace = "determined"

// Time times the duration between calling Time and calling the func() it
// returns, and observes the result using the prometheus.Observer. It can
// be used to time a function call.
// If Prometheus is disabled, it does nothing.
func Time(obs prometheus.Observer) (end func()) {
	if !config.GetMasterConfig().Observability.EnablePrometheus {
		return func() {}
	}

	start := time.Now()
	return func() {
		obs.Observe(time.Since(start).Seconds())
	}
}

// ErrCount increments the counter if the err is non-nil.
// If Prometheus is disabled, it does nothing.
func ErrCount(counter prometheus.Counter, err *error) {
	if !config.GetMasterConfig().Observability.EnablePrometheus {
		return
	}

	if err != nil {
		counter.Inc()
	}
}
