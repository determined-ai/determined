package api

import (
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/logger"
)

/* Shared types? */
type LogStreamRequest struct {
	Offset int
	Limit  int
	Follow bool
}

type ServerSend func(logger.Entry) error

func ProcessLogs(req LogStreamRequest,
	eventMgrAddr actor.Address,
	system *actor.System,
	cb ServerSend,
) error {

	logEntries := make([]logger.Entry, 0)
	for {
		err := ActorRequest(system, eventMgrAddr, req, &logEntries)

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
			time.Sleep(2000 * time.Millisecond)
		}
		if !req.Follow || req.Limit == 0 {
			return nil
		} else if req.Follow {
			req.Limit = -1
		}
	}
}
