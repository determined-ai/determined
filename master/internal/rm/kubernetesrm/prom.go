package kubernetesrm

import (
	"time"

	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/determined-ai/determined/master/internal/config"
)

const (
	promNamespace = "determined"
	promSubsystem = "kubernetesrm"

	promGoFuncLabel = "go_func"
	// promKubeAPILabel = "kube_apiserver".
)

var (
	k8sPromLabels    = []string{"target", "method", "resource"}
	k8sPromHistogram = prom.NewHistogramVec(prom.HistogramOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "seconds",
		Help:      "duration of kubernetesrm internals",
		Buckets:   prom.DefBuckets,
	}, k8sPromLabels)
)

func init() {
	prom.MustRegister(k8sPromHistogram)
}

func recordK8sTiming(labels ...string) (end func()) {
	if !config.GetMasterConfig().Observability.EnablePrometheus {
		return func() {}
	}

	start := time.Now()
	return func() {
		k8sPromHistogram.WithLabelValues(labels...).Observe(time.Since(start).Seconds())
	}
}
