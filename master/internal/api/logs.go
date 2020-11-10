package api

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// LogsRequest describes the parameters needed to target a subset of logs.
type LogsRequest struct {
	Offset  int
	Limit   int
	Follow  bool
	Filters []Filter
}

// LogBatch represents a batch of logs.
type LogBatch interface {
	ForEach(func(interface{}) error) error
	Size() int
}

// OnLogBatchFn is a callback called on each batch of logs.
type OnLogBatchFn func(LogBatch) error

// LogFetcherFn fetches returns a batch of logs based on a LogRequest.
type LogFetcherFn func(LogsRequest) (LogBatch, error)

// TerminationCheckFn checks whether the log processing should stop or not.
type TerminationCheckFn func() (bool, error)

// LogStoreProcessor is a actor that fetches logs and processes them. It handles common
// logic around limits, offsets and backpressure.
//
// The type of batches produced by the LogFetcherFn must match those handled by the
// OnLogBatchFn or the actor will panic.
type LogStoreProcessor struct {
	ctx            context.Context
	req            LogsRequest
	fetcher        LogFetcherFn
	process        OnLogBatchFn
	terminateCheck TerminationCheckFn
	batchWaitTime  *time.Duration
}

// NewLogStoreProcessor creates a new LogStoreProcessor.
func NewLogStoreProcessor(
	ctx context.Context,
	req LogsRequest,
	fetcher LogFetcherFn,
	process OnLogBatchFn,
	terminateCheck TerminationCheckFn,
	batchWaitTime *time.Duration,
) *LogStoreProcessor {
	return &LogStoreProcessor{
		ctx:            ctx,
		req:            req,
		fetcher:        fetcher,
		process:        process,
		terminateCheck: terminateCheck,
		batchWaitTime:  batchWaitTime,
	}
}

// Receive implements the actor.Actor interface.
func (l *LogStoreProcessor) Receive(ctx *actor.Context) error {
	type tick struct{}
	switch ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self(), tick{})

	case tick:
		if l.ctx.Err() != nil {
			ctx.Self().Stop()
			return nil
		}

		defer func() {
			if l.batchWaitTime != nil {
				actors.NotifyAfter(ctx, *l.batchWaitTime, tick{})
			} else {
				ctx.Tell(ctx.Self(), tick{})
			}
		}()

		switch batch, err := l.fetcher(l.req); {
		case err != nil:
			return errors.Wrapf(err, "failed to fetch logs")
		case batch == nil, batch.Size() == 0:
			if !l.req.Follow {
				ctx.Self().Stop()
				return nil
			}

			if l.terminateCheck != nil {
				terminate, err := l.terminateCheck()
				switch {
				case err != nil:
					return errors.Wrap(err, "failed to check the termination status.")
				case terminate:
					ctx.Self().Stop()
					return nil
				}
			}
		default:
			l.req.Limit -= batch.Size()
			l.req.Offset += batch.Size()
			switch err := l.process(batch); {
			case err != nil:
				return fmt.Errorf("failed while processing batch: %w", err)
			case !l.req.Follow && l.req.Limit <= 0:
				ctx.Self().Stop()
				return nil
			}
		}

	case actor.PostStop:

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// LogStreamProcessor handles streaming log messages. Upon start, it notifies another
// actor which handles the LogsRequest message to start streaming logs conforming to that
// request to itself. Each time the producing actor receives a batch, it will send it to
// the LogStreamProcessor to handle it with its OnLogBatchFn.
type LogStreamProcessor struct {
	req         LogsRequest
	ctx         context.Context
	send        OnLogBatchFn
	logStore    *actor.Ref
	sendCounter int
}

// CloseStream indicates that the log stream should close.
type CloseStream struct{}

// NewLogStreamProcessor creates a new logStreamActor.
func NewLogStreamProcessor(
	ctx context.Context,
	eventManager *actor.Ref,
	request LogsRequest,
	send OnLogBatchFn,
) *LogStreamProcessor {
	return &LogStreamProcessor{req: request, ctx: ctx, send: send, logStore: eventManager}
}

// Receive implements the actor.Actor interface.
func (l *LogStreamProcessor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		if response := ctx.Ask(l.logStore, l.req); response.Empty() {
			ctx.Self().Stop()
			return status.Errorf(codes.NotFound, "logStore did not respond")
		}

	case LogBatch:
		if l.ctx.Err() != nil {
			ctx.Self().Stop()
			break
		}
		if err := l.send(msg); err != nil {
			return status.Errorf(codes.Internal, "failed to send batch starting at %d", l.sendCounter)
		}
		l.sendCounter += msg.Size()
		if l.req.Limit > 0 && l.sendCounter >= l.req.Limit {
			ctx.Self().Stop()
		}

	case CloseStream:
		ctx.Self().Stop()

	case actor.PostStop:
		ctx.Tell(l.logStore, CloseStream{})

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
