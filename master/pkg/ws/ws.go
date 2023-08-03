package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// pingInterval is the interval at which to send pings.
	pingInterval = 15 * time.Second
	// pongWait is the duration to wait for a pong response to a ping.
	pongWait = time.Minute
	// closeWait is the duration to wait for a close response.
	closeWait = 5 * time.Second
	// inboxBufferSize is the number of messages to read before applying backpressure.
	inboxBufferSize = 32
	// outboxBufferSize is the number of messages to write before applying backpressure.
	outboxBufferSize = 64
	// maxMessageSize is the maximum size of a websocket message that we send in bytes, copied
	// from MAX_WEBSOCKET_MSG_SIZE in determined/constants.py.
	maxMessageSize = 128 * 1024 * 1024
)

// WebSocket is a facade that wraps a Gorilla websocket and provides a higher-level, type-safe, and
// thread-safe API by specializing for JSON encoding/decoding and using channels for read/write. The
// Close method must be called or resources will be leaked.
type WebSocket[TIn, TOut any] struct {
	log  *logrus.Entry
	conn *websocket.Conn

	cancel    context.CancelFunc
	errLock   sync.Mutex
	err       error
	closeOnce sync.Once

	// Inbox is a channel for incoming messages.
	Inbox <-chan TIn
	// Outbox is a channel for outgoing messages.
	Outbox chan<- TOut
	// Done notifies when the Websocket is closed. A read on Done blocks until this condition.
	Done <-chan struct{}
}

// Wrap the given, underlying *websocket.Conn and returns a higher level, thread-safe wrapper.
func Wrap[TIn, TOut any](name string, conn *websocket.Conn) (*WebSocket[TIn, TOut], error) {
	ctx, cancel := context.WithCancel(context.Background())

	inbox := make(chan TIn, inboxBufferSize)
	outbox := make(chan TOut, outboxBufferSize)
	done := make(chan struct{})

	s := &WebSocket[TIn, TOut]{
		log: logrus.WithFields(logrus.Fields{
			"component":   "websocket",
			"remote-addr": conn.RemoteAddr(),
			"name":        name,
		}),
		conn:   conn,
		cancel: cancel,
		Inbox:  inbox,
		Outbox: outbox,
		Done:   done,
	}

	s.conn.SetReadLimit(maxMessageSize)
	if err := s.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return nil, fmt.Errorf("setting initial read deadline: %w", err)
	}
	s.conn.SetPongHandler(func(string) error {
		if err := s.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			s.log.WithError(err).Error("setting read deadline")
		}
		return nil
	})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := s.runWriteLoop(ctx, outbox); err != nil {
			s.setError(fmt.Errorf("write loop: %w", err))
		}
	}()
	go func() {
		defer wg.Done()
		if err := s.runReadLoop(ctx, inbox); err != nil {
			s.setError(fmt.Errorf("read loop: %w", err))
		}
	}()

	go func() {
		wg.Wait()
		close(done)
	}()

	return s, nil
}

func (s *WebSocket[TIn, TOut]) runReadLoop(ctx context.Context, inbox chan<- TIn) error {
	s.log.Trace("running socket read loop")
	defer s.cancel()
	defer close(inbox)

	for {
		switch msgType, msg, err := s.conn.ReadMessage(); {
		case websocket.IsCloseError(err, websocket.CloseNormalClosure):
			return nil
		case err != nil:
			return fmt.Errorf("reading message: %w", err)
		case msgType != websocket.TextMessage && msgType != websocket.BinaryMessage:
			return fmt.Errorf("unexpected message type: %d", msgType)
		default:
			if ctx.Err() != nil {
				// If canceled, drop and read until a close is read.
				continue
			}

			var parsed TIn
			if err := json.Unmarshal(msg, &parsed); err != nil {
				return fmt.Errorf("unmarshalling message: %w", err)
			}
			inbox <- parsed
		}
	}
}

func (s *WebSocket[TIn, TOut]) runWriteLoop(ctx context.Context, outbox <-chan TOut) error {
	s.log.Trace("running socket write loop")
	defer s.cancel()

	ping := time.NewTicker(pingInterval)
	defer ping.Stop()
	for {
		select {
		case msg := <-outbox:
			bs, err := json.Marshal(&msg)
			switch {
			case err != nil:
				return fmt.Errorf("encoding outbound message: %w", err)
			case len(bs) > maxMessageSize:
				return fmt.Errorf("message size %d exceeds maximum size %d", len(bs), maxMessageSize)
			}

			switch err := s.conn.WriteMessage(websocket.TextMessage, bs); {
			case err == websocket.ErrCloseSent:
				return nil
			case err != nil:
				return fmt.Errorf("writing message: %w", err)
			}
		case <-ping.C:
			err := s.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(pongWait))
			netErr, ok := err.(net.Error)
			switch {
			case ok && netErr.Timeout():
				continue
			case err == websocket.ErrCloseSent:
				return nil
			case err != nil:
				return fmt.Errorf("sending ping: %w", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// Close closes the websocket by performing the close handshake and closing the underlying
// connection, rendering it unusable.
func (s *WebSocket[TIn, TOut]) Close() error {
	s.closeOnce.Do(func() {
		s.log.Trace("attempting graceful close")
		if hErr := s.closeGraceful(); hErr != nil {
			s.setError(fmt.Errorf("gracefully closing: %w", hErr))
			s.log.Trace("attempting forceful close")
			if fErr := s.closeForced(); fErr != nil {
				s.log.WithError(fErr).Error("failed to forcibly close socket")
			}
		}
		s.log.Trace("socket closed")
	})
	return s.Error()
}

func (s *WebSocket[TIn, TOut]) closeGraceful() error {
	// https://github.com/gorilla/websocket/issues/448 suggests that you can do something like
	//
	//   s.conn.SetPongHandler(nil) // So the pong handler doesn't extend the deadline.
	//   s.conn.SetReadDeadline(closeDeadline)
	//
	// to properly enforce a close deadline, but concurrently reading and writing pong handlers
	// is a race, since handling of control frames within conn.advanceFrame uses the pong handlers.
	// The likelihood of this race mattering in practice is miniscule but the race detector sees it.
	// We enforce the deadline ourselves, outside of github.com/gorilla/websocket to avoid.
	closeDeadline := time.Now().Add(closeWait)
	s.cancel()

	// If this close message begins the handshake, the read loop will exhaust messages until our
	// peer responds with their close, or it exceeds the read deadline (or, you know, the
	// underlying connection is ripped from its hands), then exit. If we did not begin the close,
	// we must have received and responded with, by the default close handler, a close. In this
	// case, we will receive ErrCloseSent from the write and the read loop should have exited.
	if err := s.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "close called"),
		closeDeadline,
	); err != websocket.ErrCloseSent && err != nil {
		return fmt.Errorf("sending close: %w", err)
	}

	select {
	case <-time.After(closeDeadline.Sub(time.Now())):
		return fmt.Errorf("did not close within the deadline")
	case <-s.Done:
	}

	if clErr := s.conn.Close(); clErr != nil {
		return fmt.Errorf("closing underlying conn: %w", clErr)
	}
	return nil
}

func (s *WebSocket[TIn, TOut]) closeForced() error {
	s.cancel()
	if err := s.conn.Close(); err != nil {
		<-s.Done
		return fmt.Errorf("closing underlying conn: %w", err)
	}
	<-s.Done
	return nil
}

// Error returns an error if the Websocket has encountered one. Errors from closing are excluded.
func (s *WebSocket[TIn, TOut]) Error() error {
	s.errLock.Lock()
	defer s.errLock.Unlock()
	return s.err
}

func (s *WebSocket[TIn, TOut]) setError(err error) {
	s.errLock.Lock()
	defer s.errLock.Unlock()
	if s.err == nil {
		s.err = err
	}
}
