package api

import (
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/pkg/logger"
)

const logCheckWaitTime = 500 * time.Millisecond

/* Shared types? */
type LogStreamRequest struct {
	Offset int
	Limit  int
	Follow bool
}

// TODO rename?
type onLogEntry func(*logger.Entry) error
type FetchLogs func(LogStreamRequest) ([]*logger.Entry, error)

// TODO add termination condition
func ProcessLogs(req LogStreamRequest,
	logFetcher FetchLogs, // TODO a better name
	cb onLogEntry,
) error {

	if err := grpc.ValidateRequest(
		grpc.ValidateLimit(int32(req.Limit)),
	); err != nil {
		return err
	}

	for {
		logEntries, err := logFetcher(req)

		if err != nil {
			return err
		}
		fmt.Printf("got %d log enties back\n", len(logEntries))
		for _, log := range logEntries {
			req.Offset++
			req.Limit--
			if err := cb(log); err != nil {
				return err
			}
		}
		if len(logEntries) == 0 {
			time.Sleep(logCheckWaitTime)
		}
		if !req.Follow || req.Limit == 0 {
			return nil
		} else if req.Follow {
			req.Limit = -1
		}
	}
}
