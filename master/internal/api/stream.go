package api

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// BatchRequest describes the parameters needed to target a subset of logs.
type BatchRequest struct {
	Offset int
	Limit  int
	Follow bool
}

// BatchResult contains either a batch or an error.
type BatchResult struct {
	batch Batch
	err   error
}

// Batch returns the inner batch or nil.
func (r *BatchResult) Batch() Batch {
	return r.batch
}

// Err returns the inner error or nil.
func (r *BatchResult) Err() error {
	return r.err
}

// OkBatchResult returns a BatchResult with a valid batch and nil error.
func OkBatchResult(b Batch) BatchResult {
	return BatchResult{batch: b}
}

// ErrBatchResult returns a BatchResult with an error and no batch.
func ErrBatchResult(err error) BatchResult {
	return BatchResult{err: err}
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

// BatchStreamProcessor is an actor that fetches batches and processes them. It handles common
// logic around limits, offsets and backpressure.
//
// The type of batches produced by the FetchBatchFn must match those handled by the
// OnBatchFn or the actor will panic.
type BatchStreamProcessor struct {
	req                    BatchRequest
	fetcher                FetchBatchFn
	terminateCheck         TerminationCheckFn
	alwaysCheckTermination bool
	batchWaitTime          *time.Duration
	batchMissWaitTime      *time.Duration
}

// NewBatchStreamProcessor creates a new BatchStreamProcessor.
func NewBatchStreamProcessor(
	req BatchRequest,
	fetcher FetchBatchFn,
	terminateCheck TerminationCheckFn,
	alwaysCheckTermination bool,
	batchWaitTime *time.Duration,
	batchMissWaitTime *time.Duration,
) *BatchStreamProcessor {
	return &BatchStreamProcessor{
		req:                    req,
		fetcher:                fetcher,
		terminateCheck:         terminateCheck,
		alwaysCheckTermination: alwaysCheckTermination,
		batchWaitTime:          batchWaitTime,
		batchMissWaitTime:      batchMissWaitTime,
	}
}

// Run runs the batch stream processor. There is an implicit assumption upstream that errors
// won't be sent forever, so after encountering an error in Run, we should log and continue
// or send and return.
func (p *BatchStreamProcessor) Run(ctx context.Context, res chan BatchResult) {
	defer close(res)
	for {
		var miss bool
		switch batch, err := p.fetcher(p.req); {
		case err != nil:
			res <- ErrBatchResult(errors.Wrapf(err, "failed to fetch batch"))
			return
		case batch == nil, batch.Size() == 0:
			if !p.req.Follow {
				return
			}
			miss = true
		default:
			// Check the ctx again before we process, since fetch takes most of the time and
			// a send on a closed ctx will print errors in the master log that can be misleading.
			if ctx.Err() != nil {
				return
			}
			p.req.Limit -= batch.Size()
			p.req.Offset += batch.Size()
			res <- OkBatchResult(batch)
			if !p.req.Follow && p.req.Limit <= 0 {
				return
			}
		}

		if (miss || p.alwaysCheckTermination) && p.terminateCheck != nil {
			switch terminate, err := p.terminateCheck(); {
			case err != nil:
				res <- ErrBatchResult(errors.Wrap(err, "failed to check the termination status"))
				return
			case terminate:
				return
			}
		}

		switch {
		case ctx.Err() != nil:
			return
		case miss && p.batchMissWaitTime != nil:
			time.Sleep(*p.batchMissWaitTime)
		case p.batchWaitTime != nil:
			time.Sleep(*p.batchWaitTime)
		}
	}
}

// Sender represents something that can send data
type Sender interface {
	Send(interface{}) error
}

// SocketLike is a struct that can send data to a websocket or another destination, if set.
type SocketLike struct {
	*websocket.Conn
	Target Sender
}
