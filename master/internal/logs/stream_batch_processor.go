package logs

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// StreamBatchProcessor signals an actor that handles logs and the logs.LogRequest message to
// forward logs to it and receives and processes these logs.
type StreamBatchProcessor struct {
	ctx      context.Context
	req      StreamRequest
	producer *actor.Ref
	process  OnBatchFn
}

type (
	// StreamRequest tells a batch-producing actor to stream batches to the sender.
	StreamRequest struct {
		Offset int
		Limit  int
		Follow bool
	}
	// CloseStream indicates that the batch streamer should close.
	CloseStream struct{}
)

// NewStreamBatchProcessor creates a new StreamBatchProcessor that notifies another
// actor to begin streaming batches to it. The type of batches produced by the producer must
// match those handled by the OnBatchFn or the actor will panic.
func NewStreamBatchProcessor(
	ctx context.Context,
	req StreamRequest,
	producer *actor.Ref,
	process OnBatchFn,
) *StreamBatchProcessor {
	return &StreamBatchProcessor{ctx: ctx, req: req, producer: producer, process: process}
}

// Receive implements the actor.Actor interface.
func (p *StreamBatchProcessor) Receive(ctx *actor.Context) error {
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
