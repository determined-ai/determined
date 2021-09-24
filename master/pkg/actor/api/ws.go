package api

import (
	"bytes"
	"encoding/json"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// TODO: Add a write size limit.

const (
	// pingWaitDuration is the duration to wait for a pong response to a ping.
	pingWaitDuration = 1 * time.Minute
	// pingInterval is the duration to wait for between pinging connections.
	pingInterval = 1 * time.Minute
)

const (
	// MaxWebsocketMessageSize is the maximum size of a websocket message that we send in bytes.
	// This is copied from MAX_WEBSOCKET_MSG_SIZE in determined/constants.py.
	MaxWebsocketMessageSize = 128 * 1024 * 1024
)

// WebSocketConnected notifies the actor that a websocket is attempting to connect.
type WebSocketConnected struct {
	Ctx echo.Context
}

// Accept wraps the connecting websocket connection in an actor.
func (w WebSocketConnected) Accept(
	ctx *actor.Context,
	msgType interface{},
	usePing bool,
) (*actor.Ref, bool) {
	conn, err := upgrader.Upgrade(w.Ctx.Response(), w.Ctx.Request(), nil)
	if err != nil {
		ctx.Respond(errors.Wrap(err, "websocket connection error"))
		return nil, false
	}
	a, _ := ctx.ActorOf("websocket-"+uuid.New().String(), WrapSocket(conn, msgType, usePing))
	ctx.Respond(a)
	return a, true
}

// WriteMessage is a message to a websocketActor asking it to write out the
// given message, encoding it to JSON.
type WriteMessage struct {
	actor.Message
}

// WriteRawMessage is a message to a websocketActor asking it to write out the
// given message without encoding to JSON.
type WriteRawMessage struct {
	actor.Message
}

// WriteResponse is the response to a successful WriteMessage.
type WriteResponse struct{}

// WriteSocketJSON writes a JSON-serializable object to a websocket actor.
func WriteSocketJSON(ctx *actor.Context, socket *actor.Ref, msg interface{}) error {
	resp := ctx.Ask(socket, WriteMessage{
		Message: msg,
	}).Get()

	switch resp := resp.(type) {
	case error:
		return errors.WithStack(resp)
	case WriteResponse:
		return nil
	default:
		return errors.Errorf("unknown response %T: %s", resp, resp)
	}
}

// WriteSocketRaw writes a string to a websocket actor.
func WriteSocketRaw(ctx *actor.Context, socket *actor.Ref, msg interface{}) error {
	resp := ctx.Ask(socket, WriteRawMessage{
		Message: msg,
	}).Get()

	switch resp := resp.(type) {
	case error:
		return errors.WithStack(resp)
	case WriteResponse:
		return nil
	default:
		return errors.Errorf("unknown response %T: %s", resp, resp)
	}
}

// WrapSocket wraps a websocket connection as an actor.
func WrapSocket(conn *websocket.Conn, msgType interface{}, usePing bool) actor.Actor {
	return &websocketActor{
		conn:         conn,
		msgType:      reflect.TypeOf(msgType),
		usePing:      usePing,
		pendingPings: make(map[string]time.Time),
	}
}

type websocketActor struct {
	conn    *websocket.Conn
	msgType reflect.Type

	usePing      bool
	pingLock     sync.Mutex
	pendingPings map[string]time.Time
}

// Receive implements the actor.Actor interface.
func (s *websocketActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		if s.usePing {
			s.setupPingLoop(ctx)
		}
		go s.runReadLoop(ctx)
		return nil
	case actor.PostStop:
		return s.conn.Close()
	case error: // Socket read errors.
		return msg
	case []byte: // Incoming messages on the socket.
		parsed, err := parseMsg(msg, s.msgType)
		if err != nil {
			return err
		}
		// Notify the socket's parent actor of the incoming message.
		ctx.Tell(ctx.Self().Parent(), parsed)
		return nil
	case WriteMessage:
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(msg.Message); err != nil {
			return err
		}
		return s.processWriteMessage(ctx, buf)
	case WriteRawMessage:
		var buf bytes.Buffer
		if _, err := buf.WriteString(msg.Message.(string)); err != nil {
			return err
		}
		return s.processWriteMessage(ctx, buf)
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
}

func (s *websocketActor) processWriteMessage(
	ctx *actor.Context,
	buf bytes.Buffer,
) error {
	if cur, max := buf.Len(), MaxWebsocketMessageSize; cur > max {
		ctx.Respond(errors.Errorf("message size %d exceeds maximum size %d", cur, max))
		return nil
	}

	ctx.Respond(WriteResponse{})

	return s.conn.WriteMessage(websocket.TextMessage, buf.Bytes())
}

func isClosingError(err error) bool {
	return err == websocket.ErrCloseSent || websocket.IsCloseError(err, websocket.CloseNormalClosure)
}

func (s *websocketActor) setupPingLoop(ctx *actor.Context) {
	s.conn.SetPongHandler(func(data string) error {
		return s.handlePong(ctx, data)
	})
	go s.runPingLoop(ctx)
}

func (s *websocketActor) handlePong(ctx *actor.Context, id string) error {
	now := time.Now()

	s.pingLock.Lock()
	defer s.pingLock.Unlock()

	if deadline, ok := s.pendingPings[id]; !ok {
		ctx.Log().Errorf("unknown ping %s", id)
		return nil
	} else if deadline.Before(now) {
		// This is a ping timeout, but let checkPendingPings report the error.
		return nil
	}
	delete(s.pendingPings, id)
	return nil
}

func (s *websocketActor) checkPendingPings() error {
	now := time.Now()

	s.pingLock.Lock()
	defer s.pingLock.Unlock()

	var errs []error
	for id, deadline := range s.pendingPings {
		if deadline.Before(now) {
			errs = append(errs, errors.Errorf("ping %s did not receive pong response by %s", id, deadline))
			delete(s.pendingPings, id)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (s *websocketActor) ping() error {
	s.pingLock.Lock()
	defer s.pingLock.Unlock()

	// According to the websocket specification [1], endpoints only have to acknowledge the most
	// recent ping message, so avoid having more than one outstanding ping request.
	//
	// [1] https://tools.ietf.org/html/rfc6455#section-5.5.3
	if len(s.pendingPings) > 0 {
		return nil
	}

	id := uuid.New().String()

	deadline := time.Now().Add(pingWaitDuration)
	err := s.conn.WriteControl(websocket.PingMessage, []byte(id), deadline)
	if e, ok := err.(net.Error); ok && e.Temporary() {
		return nil
	} else if err != nil {
		return err
	}

	s.pendingPings[id] = deadline

	return nil
}

func (s *websocketActor) runPingLoop(ctx *actor.Context) {
	pingAndWait := func() error {
		if err := s.ping(); err != nil {
			return err
		}

		t := time.NewTimer(pingInterval)
		defer t.Stop()
		<-t.C
		return nil
	}

	defer ctx.Self().Stop()

	for {
		if err := s.checkPendingPings(); err != nil {
			ctx.Tell(ctx.Self(), err)
			return
		}

		if err := pingAndWait(); isClosingError(err) {
			return
		} else if err != nil {
			ctx.Tell(ctx.Self(), err)
			return
		}
	}
}

func (s *websocketActor) runReadLoop(ctx *actor.Context) {
	read := func() ([]byte, error) {
		msgType, msg, err := s.conn.ReadMessage()
		if err != nil {
			return nil, err
		}
		if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
			return nil, errors.Errorf("unexpected message type: %d", msgType)
		}
		return msg, nil
	}

	defer ctx.Self().Stop()

	for {
		msg, err := read()
		if isClosingError(err) {
			return
		} else if err != nil {
			// Socket read errors are sent to the socket actor rather than the parent. Exceptions
			// will bubble up the parent through the actor system.
			ctx.Tell(ctx.Self(), err)
			return
		}
		ctx.Tell(ctx.Self(), msg)
	}
}

func parseMsg(raw []byte, msgType reflect.Type) (interface{}, error) {
	var parsed interface{}
	if msgType.Kind() == reflect.Ptr {
		parsed = reflect.New(msgType.Elem()).Interface()
	} else {
		parsed = reflect.New(msgType).Interface()
	}
	if err := json.Unmarshal(raw, parsed); err != nil {
		return nil, err
	}

	if msgType.Kind() == reflect.Ptr {
		return parsed, nil
	}
	return reflect.ValueOf(parsed).Elem().Interface(), nil
}
