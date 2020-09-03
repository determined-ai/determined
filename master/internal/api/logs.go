package api

import (
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/pkg/errors"
)

const logCheckWaitTime = 500 * time.Millisecond

/* Shared types? */
type LogStreamRequest struct {
	Offset int
	Limit  int
	Follow bool
}

// TODO rename? or define inline
type onLogEntry func(*logger.Entry) error
type FetchLogs func(LogStreamRequest) ([]*logger.Entry, error)
type ShouldTerminateCheck func() (bool, error)

// TODO add termination condition
func ProcessLogs(req LogStreamRequest,
	logFetcher FetchLogs, // TODO a better name
	cb onLogEntry,
	terminateCheck *ShouldTerminateCheck,
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
				return errors.Wrapf(err, "onLogEntry callback failed on entry %v", log)
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
