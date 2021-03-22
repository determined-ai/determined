package api

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// BatchRequest describes the parameters needed to target a subset of logs.
type BatchRequest struct {
	Offset int
	Limit  int
	Follow bool
}

// Batch represents a batch of logs.
type Batch interface {
	ForEach(func(interface{}) error) error
	Size() int
}

// BatchOfOne is a wrapper for a single log that implements Batch.
type BatchOfOne struct {
	Inner interface{}
}

// ToBatchOfOne wraps a single entry as a BatchOfOne that implements Batch.
func ToBatchOfOne(x interface{}) BatchOfOne {
	return BatchOfOne{x}
}

// ForEach implements Batch.
func (l BatchOfOne) ForEach(f func(interface{}) error) error {
	return f(l.Inner)
}

// Size implements Batch.
func (l BatchOfOne) Size() int {
	return 1
}

// OnBatchFn is a callback called on each batch.
type OnBatchFn func(Batch) error

// FetchBatchFn fetches returns a batch.
type FetchBatchFn func(BatchRequest) (Batch, error)

// TerminationCheckFn checks whether the log processing should stop or not.
type TerminationCheckFn func() (bool, error)

// BatchStreamProcessor is a actor that fetches batches and processes them. It handles common
// logic around limits, offsets and backpressure.
//
// The type of batches produced by the FetchBatchFn must match those handled by the
// OnBatchFn or the actor will panic.
type BatchStreamProcessor struct {
	req            BatchRequest
	fetcher        FetchBatchFn
	process        OnBatchFn
	terminateCheck TerminationCheckFn
	batchWaitTime  time.Duration
}

// NewBatchStreamProcessor creates a new BatchStreamProcessor.
func NewBatchStreamProcessor(
	req BatchRequest,
	fetcher FetchBatchFn,
	process OnBatchFn,
	terminateCheck TerminationCheckFn,
	batchWaitTime time.Duration,
) *BatchStreamProcessor {
	return &BatchStreamProcessor{
		req:            req,
		fetcher:        fetcher,
		process:        process,
		terminateCheck: terminateCheck,
		batchWaitTime:  batchWaitTime,
	}
}

// Run runs the batch stream processor.
func (p *BatchStreamProcessor) Run(ctx context.Context) error {
	t := time.NewTicker(p.batchWaitTime)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			switch batch, err := p.fetcher(p.req); {
			case err != nil:
				return errors.Wrapf(err, "failed to fetch batch")
			case batch == nil, batch.Size() == 0:
				if !p.req.Follow {
					return nil
				}

				if p.terminateCheck != nil {
					terminate, err := p.terminateCheck()
					switch {
					case err != nil:
						return errors.Wrap(err, "failed to check the termination status.")
					case terminate:
						return nil
					}
				}
			default:
				// Check the ctx again before we process, since fetch takes most of the time and
				// a send on a closed ctx will print errors in the master log that can be misleading.
				if ctx.Err() != nil {
					return nil
				}
				p.req.Limit -= batch.Size()
				p.req.Offset += batch.Size()
				switch err := p.process(batch); {
				case err != nil:
					return fmt.Errorf("failed while processing batch: %w", err)
				case !p.req.Follow && p.req.Limit <= 0:
					return nil
				}
			}
		}
	}
}

// LogStreamProcessor handles streaming log messages. Upon start, it notifies another
// actor which handles the BatchRequest message to start streaming logs conforming to that
// request to itself. Each time the producing actor receives a batch, it will send it to
// the LogStreamProcessor to handle it with its OnBatchFn.
type LogStreamProcessor struct {
	req         BatchRequest
	ctx         context.Context
	send        OnBatchFn
	logStore    *actor.Ref
	sendCounter int
}

// CloseStream indicates that the log stream should close.
type CloseStream struct{}

// NewLogStreamProcessor creates a new logStreamActor.
func NewLogStreamProcessor(
	ctx context.Context,
	eventManager *actor.Ref,
	request BatchRequest,
	send OnBatchFn,
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

	case Batch:
		if connectionIsClosed(l.ctx) {
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

func connectionIsClosed(ctx context.Context) bool {
	return ctx.Err() != nil
}
