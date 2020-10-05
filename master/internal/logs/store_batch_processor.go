package logs

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

// StoreBatchProcessor is a batch processor that fetches logs and processes them with an OnBatchFn.
// It handles common logic around limits, offsets and backpressure. The type of batches produced by
// the Fetcher must match those handled by the OnBatchFn or the actor will panic.
type StoreBatchProcessor struct {
	ctx            context.Context
	limit          int
	follow         bool
	fetcher        Fetcher
	process        OnBatchFn
	terminateCheck *TerminationCheckFn
	batchWaitTime  *time.Duration
}

// NewStoreBatchProcessor creates a new StoreBatchProcessor.
func NewStoreBatchProcessor(
	ctx context.Context,
	limit int,
	follow bool,
	fetcher Fetcher,
	process OnBatchFn,
	terminateCheck *TerminationCheckFn,
	batchWaitTime *time.Duration,
) *StoreBatchProcessor {
	return &StoreBatchProcessor{
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
func (p *StoreBatchProcessor) Receive(ctx *actor.Context) error {
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
			actors.NotifyAfter(ctx, time.Second, tick{})
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
