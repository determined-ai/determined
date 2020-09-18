package api

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/proto/pkg/logv1"
)

const logCheckWaitTime = 500 * time.Millisecond

// LogsRequest describes the parameters needed to target a subset of logs.
type LogsRequest struct {
	Offset int
	Limit  int
	Follow bool
}

// OnLogEntry is a callback called on each log entry.
type OnLogEntry func(*logger.Entry) error

// LogFetcherFn fetchs returns a subset of logs based on a LogRequest.
type LogFetcherFn func(LogsRequest) ([]*logger.Entry, error)

// TerminationCheck checks whether a log processing should stop or not.
type TerminationCheck func() (bool, error)

// LogEntryToProtoLogEntry turns a logger.LogEntry into logv1.LogEntry.
func LogEntryToProtoLogEntry(logEntry *logger.Entry) *logv1.LogEntry {
	return &logv1.LogEntry{Id: int32(logEntry.ID), Message: logEntry.Message}
}

// ProcessLogs handles fetching and processing logs from a log store.
func ProcessLogs(ctx context.Context,
	req LogsRequest,
	logFetcher LogFetcherFn, // TODO a better name
	cb OnLogEntry,
	terminateCheck *TerminationCheck,
) error {
	for {
		logEntries, err := logFetcher(req)

		if err != nil {
			return errors.Wrapf(err, "failed to fetch logs for %v", req)
		}
		for _, log := range logEntries {
			req.Offset++
			req.Limit--
			if err := cb(log); err != nil {
				return errors.Wrapf(err, "failed to process log entry %v", log)
			}
		}
		if len(logEntries) == 0 {
			if err := ctx.Err(); err != nil {
				// context is closed
				return nil
			}
			if terminateCheck != nil {
				terminate, err := (*terminateCheck)()
				switch {
				case err != nil:
					return errors.Wrap(err, "failed to check the termination status.")
				case terminate:
					return nil
				}
			}
			time.Sleep(logCheckWaitTime)
		}
		if !req.Follow || req.Limit == 0 {
			return nil
		} else if req.Follow {
			req.Limit = -1
		}
	}
}

/* Commands */

// CommandLogStreamActor handles streaming log messages for commands.
type CommandLogStreamActor struct {
	req          LogsRequest
	ctx          context.Context
	send         OnLogEntry
	eventManager *actor.Ref
}

// CloseStream indicates that the log stream should close.
type CloseStream struct{}

// NewCommandLogStreamActor creates a new command logStreamActor.
func NewCommandLogStreamActor(
	ctx context.Context,
	eventManager *actor.Ref,
	request LogsRequest,
	send OnLogEntry,
) *CommandLogStreamActor {
	return &CommandLogStreamActor{req: request, ctx: ctx, send: send, eventManager: eventManager}
}

// Receive implements the actor.Actor interface.
func (l *CommandLogStreamActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		if response := ctx.Ask(l.eventManager, l.req); response.Empty() {
			ctx.Self().Stop()
			return status.Errorf(codes.NotFound, "event manager did not respond")
		}

	case logger.Entry:
		// Make sure the context is still open.
		if l.ctx.Err() != nil {
			ctx.Self().Stop()
			break
		}
		if err := l.send(&msg); err != nil {
			ctx.Self().Stop()
			return status.Errorf(codes.Internal, "failed to send log message")
		}

	case CloseStream:
		ctx.Self().Stop()

	case actor.PostStop:
		ctx.Tell(l.eventManager, CloseStream{})

	default:
		return status.Errorf(codes.Internal, fmt.Sprintf("received unsupported message %v", msg))
	}
	return nil
}
