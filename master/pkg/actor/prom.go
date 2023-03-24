package actor

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/determined-ai/determined/master/internal/config"
)

const (
	promNamespace = "determined"
	promSubsystem = "actors"
)

var (
	receiveLabels    = []string{"from", "to", "msg"}
	receiveHistogram = prom.NewHistogramVec(prom.HistogramOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "receive",
		Help:      "timings for actor receive calls",
		Buckets:   prom.DefBuckets,
	}, receiveLabels)
	receiveErrors = prom.NewCounterVec(prom.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "receive_errors",
		Help:      "errors from actor receive calls",
	}, receiveLabels)
)

func init() {
	prom.MustRegister(receiveHistogram)
	prom.MustRegister(receiveErrors)
}

func (r *Ref) time(ctx *Context) (end func()) {
	if !config.GetMasterConfig().Observability.EnablePrometheus {
		return func() {}
	}

	start := time.Now()
	return func() {
		receiveHistogram.WithLabelValues(r.promLabels(ctx)...).Observe(time.Since(start).Seconds())
	}
}

func (r *Ref) recordErr(ctx *Context) func(error) {
	if !config.GetMasterConfig().Observability.EnablePrometheus {
		return func(error) {}
	}

	return func(err error) {
		if err == nil {
			return
		}
		receiveErrors.WithLabelValues(r.promLabels(ctx)...).Inc()
	}
}

func (r *Ref) promLabels(ctx *Context) []string {
	from := "system"
	if ctx != nil && ctx.sender != nil {
		from = actorToGenericName(ctx.sender.actor)
	}

	to := actorToGenericName(r.actor)
	msg := "PreStart"
	if ctx != nil {
		msg = reflect.TypeOf(ctx.message).String()
	}
	return []string{from, to, msg}
}

func actorToGenericName(actor Actor) string {
	out := fmt.Sprintf("%T", actor)
	out = strings.TrimPrefix(out, "*")
	return out
}
