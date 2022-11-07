package internal

import (
	"fmt"
)

// MasterConnectionError is returned by the agent when the websocket connection to the master fails.
type MasterConnectionError struct {
	cause error
}

func (e MasterConnectionError) Error() string {
	return fmt.Sprintf("crashing due to master websocket connection failure: %s", e.cause)
}
