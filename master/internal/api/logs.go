package api

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/pkg/errors"
)

// Record represents a single record in a batch.
type Record interface{}

// Batch represents a batch of logs.
type Batch interface {
	ForEach(func(Record) error) error
	Size() int
}

// Fetcher is an interface for an intermediary between a batch store and a consumer.
// Fetchers accept requests to read a batch of sequential records and responds with
// a batch of size limit to its internal max batch size.
//
// Since the batch store defines the filters that are valid for it, the fetcher must
// validate the filters. This is usually done on creation.
type Fetcher interface {
	Fetch(limit int, unlimited bool) (Batch, error)
}

// OnBatchFn is a callback called on each batch of log entries.
// It returns an error and how many records were processed in the batch.
type OnBatchFn func(Batch) error

// TerminationCheckFn checks whether the log processing should stop or not.
type TerminationCheckFn func() (bool, error)

// LogStoreProcessor is a log processor that fetches logs and processes them with an OnBatchFn.
// It handles common logic around limits, offsets and backpressure. The type of batches produced by
// the Fetcher must match those handled by the OnBatchFn or the actor will panic.
type LogStoreProcessor struct {
	ctx            context.Context
	limit          int
	follow         bool
	fetcher        Fetcher
	process        OnBatchFn
	terminateCheck *TerminationCheckFn
	batchWaitTime  *time.Duration
}

// NewLogStoreProcessor creates a new LogStoreProcessor.
func NewLogStoreProcessor(
	ctx context.Context,
	limit int,
	follow bool,
	fetcher Fetcher,
	process OnBatchFn,
	terminateCheck *TerminationCheckFn,
	batchWaitTime *time.Duration,
) *LogStoreProcessor {
	return &LogStoreProcessor{
		ctx:            ctx,
		limit:          limit,
		follow:         follow,
		fetcher:        fetcher,
		process:        process,
		terminateCheck: terminateCheck,
		batchWaitTime:  batchWaitTime,
	}
}

type (
	tick struct{}
)

// Receive implements the actor.Actor interface.
func (p *LogStoreProcessor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self(), tick{})

	case tick:
		if p.ctx.Err() != nil {
			ctx.Self().Stop()
			return nil
		}

		switch batch, err := p.fetcher.Fetch(p.limit, p.follow); {
		case err != nil:
			return errors.Wrapf(err, "failed to fetch logs")
		case batch.Size() == 0:
			if p.terminateCheck != nil {
				terminate, err := (*p.terminateCheck)()
				switch {
				case err != nil:
					return errors.Wrap(err, "failed to check the termination status.")
				case terminate:
					ctx.Self().Stop()
					return nil
				}
			}
			actors.NotifyAfter(ctx, 100*time.Millisecond, tick{})
			return nil
		default:
			p.limit -= batch.Size()
			switch err := p.process(batch); {
			case err != nil:
				return fmt.Errorf("failed while processing batch: %w", err)
			case !p.follow && p.limit <= 0:
				ctx.Self().Stop()
				return nil
			}

			if p.batchWaitTime != nil {
				actors.NotifyAfter(ctx, *p.batchWaitTime, tick{})
			} else {
				ctx.Tell(ctx.Self(), tick{})
			}
		}

	case actor.PostStop:

	default:
		return status.Errorf(codes.Internal, fmt.Sprintf("received unsupported message %v", msg))
	}
	return nil
}

// LogStreamProcessor signals an actor that handles logs and the logs.LogsRequest message to
// forward logs to it and receives and processes these logs.
type LogStreamProcessor struct {
	ctx      context.Context
	req      LogsRequest
	producer *actor.Ref
	process  OnBatchFn
}

type (
	// LogsRequest tells a batch-producing actor to stream batches to the sender.
	LogsRequest struct {
		Offset int
		Limit  int
		Follow bool
	}
	// CloseStream indicates that the batch streamer should close.
	CloseStream struct{}
)

// NewLogStreamProcessor creates a new LogStreamProcessor that notifies another
// actor to begin streaming batches of logs to it. The type of batches produced by the producer must
// match those handled by the OnBatchFn or the actor will panic.
func NewLogStreamProcessor(
	ctx context.Context,
	req LogsRequest,
	producer *actor.Ref,
	process OnBatchFn,
) *LogStreamProcessor {
	return &LogStreamProcessor{ctx: ctx, req: req, producer: producer, process: process}
}

// Receive implements the actor.Actor interface.
func (p *LogStreamProcessor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		if response := ctx.Ask(p.producer, p.req); response.Empty() {
			ctx.Self().Stop()
			return status.Errorf(codes.NotFound, "producer did not respond")
		}

	case Batch:
		if p.ctx.Err() != nil {
			ctx.Self().Stop()
			return nil
		}
		p.req.Limit -= msg.Size()
		switch err := p.process(msg); {
		case err != nil:
			return fmt.Errorf("failed while processing batch: %w", err)
		case !p.req.Follow && p.req.Limit <= 0:
			ctx.Self().Stop()
			return nil
		}

	case CloseStream:
		ctx.Self().Stop()

	case actor.PostStop:
		ctx.Tell(p.producer, CloseStream{})

	default:
		return status.Errorf(codes.Internal, fmt.Sprintf("received unsupported message %v", msg))
	}
	return nil
}
