package stream

import (
	"fmt"

	"github.com/gorilla/websocket"
)

type WebsocketLike interface {
	ReadJSON(interface{}) error
	Write(interface{}) error
	Close() error
}

type WrappedWebsocket struct {
	*websocket.Conn
}

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
