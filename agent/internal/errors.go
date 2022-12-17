package internal

import (
	"fmt"
)

// longDisconnected is returned by the agent when the websocket connection to the master fails.
type longDisconnected struct {
	cause error
}

func (e longDisconnected) Error() string {
	return fmt.Sprintf("long disconnected, and unable to reconnect, from master: %s", e.cause)
}
