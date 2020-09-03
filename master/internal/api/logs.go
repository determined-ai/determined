package api

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/grpc"
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
func ProcessLogs(req LogsRequest,
	logFetcher LogFetcher, // TODO a better name
	cb onLogEntry,
	terminateCheck *TerminationCheck,
) error {
	// DISCUSS should this be left out to the caller? in some cases they can't leave it until this fn
	// call
	if err := grpc.ValidateRequest(
		grpc.ValidateLimit(int32(req.Limit)),
	); err != nil {
		return err
	}

	for {
		logEntries, err := logFetcher(req)

		if err != nil {
			return errors.Wrapf(err, "failed to fetch logs for %v", req)
		}
		fmt.Printf("got %d log enties back\n", len(logEntries))
		for _, log := range logEntries {
			req.Offset++
			req.Limit--
			if err := cb(log); err != nil {
				return errors.Wrapf(err, "failed to process log entry %v", log)
			}
		}
		if len(logEntries) == 0 {
			if terminateCheck != nil {
				terminate, err := (*terminateCheck)()
				if err != nil {
					return errors.Wrap(err, "failed to check the termination status.")
				}
				if terminate {
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
