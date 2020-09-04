package api

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

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

// TODO rename? or define inline
type onLogEntry func(*logger.Entry) error

// LogFetcher fetchs returns a subset of logs based on a LogRequest.
type LogFetcher func(LogsRequest) ([]*logger.Entry, error)

// TerminationCheck checks whether a log processing should stop or not.
type TerminationCheck func() (bool, error)

// ProcessLogs handles fetching and processing logs from a log store.
func ProcessLogs(ctx context.Context,
	req LogsRequest,
	logFetcher LogFetcher, // TODO a better name
	cb onLogEntry,
	terminateCheck *TerminationCheck,
) error {
	// FIXME how does it terminate when the caller goes away
	for {
		fmt.Printf("sending log request %v. ", req)
		logEntries, err := logFetcher(req)
		fmt.Printf("received %d logs.\n", len(logEntries))

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

// LogEntryToProtoLogEntry turns a logger.LogEntry into logv1.LogEntry.
func LogEntryToProtoLogEntry(logEntry *logger.Entry) *logv1.LogEntry {
	return &logv1.LogEntry{Id: int32(logEntry.ID), Message: logEntry.Message}
}
