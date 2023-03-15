package actor

import (
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/determined-ai/determined/master/internal/config"
)

const (
	promNamespace = "determined"
	promSubsystem = "actors"
	wildcard      = "*"
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
		from = r.normalizeAddr(ctx.sender.address.path)
	}
	to := r.normalizeAddr(r.address.path)
	msg := "PreStart"
	if ctx != nil {
		msg = reflect.TypeOf(ctx.message).String()
	}
	return []string{from, to, msg}
}

// normalizeAddr exists to normalize actor paths like /trials/1 and /noisy-actor-xyz into
// /trials/* and /noisy-actor-* so that there isn't an explosion of prometheus labels.
func (r *Ref) normalizeAddr(addr string) string {
	out := "/"
	for _, part := range filepath.SplitList(addr) {
		if _, err := uuid.Parse(part); err == nil {
			part = wildcard
		} else if _, err := strconv.Atoi(part); err == nil {
			part = wildcard
		} else {
			for _, noisy := range noisyActors {
				if strings.Contains(part, noisy) {
					part = part[:len(noisy)] + wildcard
					break
				}
			}
		}
		out = filepath.Join(out, part)
	}
	return out
}
