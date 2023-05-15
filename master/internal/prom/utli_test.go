package prom_test

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/determined-ai/determined/master/internal/prom"
)

var (
	labels    = []string{"method"}
	histogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: prom.DeterminedNamespace,
		Subsystem: "my-subsystem",
		Name:      "seconds",
		Buckets:   prometheus.DefBuckets,
	}, labels)
)

func ExampleTime() {
	defer prom.Time(histogram.WithLabelValues("GET"))

	// do thing you want to time.
	time.Sleep(time.Millisecond)
	// Output:
}

var counter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: prom.DeterminedNamespace,
	Subsystem: "my-subsystem",
	Name:      "errors",
}, labels)

func ExampleErrCount() {
	var err error
	defer prom.ErrCount(counter.WithLabelValues("GET"), &err)

	// do some stuff that may cause error to be non-nil
	_, err = strconv.Atoi("abc")
	// Output:
}
