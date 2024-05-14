package stream

import (
	"fmt"

	"github.com/gorilla/websocket"
)

// WebsocketLike is an interface to describe a websocket and its dummy implementations.
type WebsocketLike interface {
	ReadJSON(interface{}) error
	Write(interface{}) error
}

// WrappedWebsocket is a simple wrapper on a websocket connection.
type WrappedWebsocket struct {
	*websocket.Conn
}

// Write ensures the underlying msg is a websocket.PreparedMessage before writing it to websocket.
func (w *WrappedWebsocket) Write(msg interface{}) error {
	pm, ok := msg.(*websocket.PreparedMessage)
	if !ok {
		return fmt.Errorf("received message that is not a prepared message")
	}
	err := w.WritePreparedMessage(pm)
	if err != nil {
		return err
	}
	return nil
}

// WriteAll attempts to write all provided messages.
func WriteAll(socketLike WebsocketLike, msgs []interface{}) error {
	for _, msg := range msgs {
		err := socketLike.Write(msg)
		if err != nil {
			return err
		}
	}
	return nil
}
