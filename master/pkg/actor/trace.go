package actor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/uber/jaeger-client-go/config"
)

// traceEnabled configs actors to submit traces to an opentracing backend (specifically Jaeger).
var traceEnabled = traceEnabledStr == "true"
var traceEnabledStr = "true"

const (
	askOperation = "Ask"
	tellOperation = "Tell"
)

// traceSend traces a send to another actor.
func traceSend(
	ctx context.Context, sender, recipient *Ref, message Message, op string,
) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if isNoisy(sender) {
		return ctx
	}
	var span opentracing.Span
	if sender == nil {
		sender = recipient.System().Ref
	}
	span, ctx = opentracing.StartSpanFromContextWithTracer(ctx, sender.tracing.tracer,
		fmt.Sprintf("%s %s %T", op, recipient.Address().path, message))
	span.Finish()
	return ctx
}

// traceReceive traces an actor's Receive call.
func traceReceive(aContext *Context, r *Ref) func() {
	if isNoisy(aContext.Self()) {
		return func() {}
	}
	span, ctx := opentracing.StartSpanFromContextWithTracer(aContext.inner,
		r.tracing.tracer, fmt.Sprintf("Handling %T", aContext.message))
	aContext.inner = ctx
	return func() {
		if r.err != nil {
			span.LogFields(otlog.String("error", r.err.Error()))
		}
		span.Finish()
	}
}

// addTracer adds a opentracing.Tracer to the actor.Ref and returns it.
func addTracer(ref *Ref) {
	hostname, err := os.Hostname()
	if err != nil {
		ref.log.WithError(err).Error(
			"failed to retrieve hostname for tracer")
	}
	tracer, closer, err := initJaeger(
		fmt.Sprintf("%s-%s%s", hostname, filepath.Base(os.Args[0]), ref.Address().path))
	if err != nil {
		ref.log.WithError(err).Error("failed to init tracer")
	}
	ref.tracing.tracer = tracer
	ref.tracing.closer = closer
}

// closeTracer closes all tracing resources associated with the actor.Ref.
func closeTracer(ref *Ref) {
	err := ref.tracing.closer.Close()
	if err != nil {
		ref.log.WithError(err).Warn("failed to close tracer")
	}
	ref.tracing.tracer = nil
	ref.tracing.closer = nil
}

// initJaeger returns an instance of Jaeger Tracer that samples 100% of traces.
func initJaeger(service string) (opentracing.Tracer, io.Closer, error) {
	tracer, closer, err := config.Configuration{
		ServiceName: service,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}.NewTracer()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to init Jaeger")
	}
	return tracer, closer, nil
}

// Actors that are too noisy to be worth tracing.
var noisyActors = []string{
	"notify-timer-",
}

func isNoisy(sender *Ref) bool {
	if sender == nil {
		return false
	}
	for _, noisyActor := range noisyActors {
		if strings.Contains(sender.Address().path, noisyActor) {
			return true
		}
	}
	return false
}
