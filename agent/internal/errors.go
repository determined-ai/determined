package internal

import "fmt"

// masterConnectionError is returned by the agent when the websocket connection to the master fails.
type masterConnectionError struct {
	cause error
}

func (e masterConnectionError) Unwrap() error {
	return e.cause
}

func (e masterConnectionError) Error() string {
	return fmt.Sprintf("long disconnected and unable to reconnect to master: %s", e.cause.Error())
}
