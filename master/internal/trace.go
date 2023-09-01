package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/uber/jaeger-client-go/config"
)

func TraceCall(ctx *context.Context, req *apiv1.GetExperimentsRequest) {
	var span opentracing.Span
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil {
		return
	}

	hostname, err := os.Hostname()
	if err != nil {
		return
	}

	service := fmt.Sprintf("%s-%s", hostname, filepath.Base(os.Args[0]))
	tracer, closer, err := initJaeger(service)
	defer closeTracer(tracer, closer)
	if err != nil {
		return
	}

	span, ctx = opentracing.StartSpanFromContextWithTracer(ctx, tracer,
		fmt.Sprint("Calling GET Experiments"))
	span.Finish()

}

// closeTracer closes all tracing resources.
func closeTracer(tracer *opentracing.Tracer, closer *io.Closer) error {
	err := closer.Close()
	if err != nil {
		return err
	}
	tracer = nil
	closer = nil
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
