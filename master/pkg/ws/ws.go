package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-multierror"
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

// Websocket provides a higher level websocket library on top of "github.com/gorilla/websocket".
type Websocket[TIn, TOut any] struct {
	// System dependencies.
	log  *logrus.Entry
	conn *websocket.Conn

	// Internal state.
	cancel    context.CancelFunc
	errLock   sync.Mutex
	err       error
	closeOnce sync.Once
	closeErr  error
	// Done signals the websocket is finished when it is closed. Even if it has exited, call Close.
	Done <-chan struct{}
	// Inbox is a channel for incoming messages.
	Inbox <-chan TIn
	// Outbox is a channel for outgoing messages.
	Outbox chan<- TOut
}

// Wrap the given, underlying *websocket.Conn and returns a higher level, thread-safe wrapper.
func Wrap[TIn, TOut any](name string, conn *websocket.Conn) *Websocket[TIn, TOut] {
	ctx, cancel := context.WithCancel(context.Background())

	inbox := make(chan TIn, inboxBufferSize)
	outbox := make(chan TOut, outboxBufferSize)
	done := make(chan struct{})

	s := &Websocket[TIn, TOut]{
		log: logrus.WithFields(logrus.Fields{
			"component":   "websocket",
			"remote-addr": conn.RemoteAddr(),
			"name":        name,
		}),
		conn: conn,

		cancel: cancel,
		Done:   done,
		Inbox:  inbox,
		Outbox: outbox,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.runWriteLoop(ctx, outbox); err != nil {
			s.setError(fmt.Errorf("write loop: %w", err))
		}
	}()

	wg.Add(1)
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

	return s
}

func (s *Websocket[TIn, TOut]) Wait() error {
	<-s.Done
	return s.Error()
}

// Error returns an error is the websocket has encountered one. Errors from closing are excluded.
func (s *Websocket[TIn, TOut]) Error() error {
	s.errLock.Lock()
	defer s.errLock.Unlock()
	return s.err
}

// Close closes the websocket by performing the close handshake and closing the underlying
// connection, rendering it unusable.
func (s *Websocket[TIn, TOut]) Close() error {
	s.closeOnce.Do(func() {
		initialErr := s.Error()

		var err *multierror.Error
		s.log.Trace("attempting graceful close")
		if hErr := s.closeGraceful(); hErr != nil {
			err = multierror.Append(err, fmt.Errorf("gracefully closing: %w", hErr))
			s.log.Trace("attempting forceful close")
			if fErr := s.closeForced(); fErr != nil {
				err = multierror.Append(err, fmt.Errorf("forcibly closing: %w", hErr))
			}
		}
		s.log.Trace("socket closed")

		if endingErr := s.Error(); initialErr == nil && endingErr != nil {
			err = multierror.Append(err, endingErr)
		}

		s.closeErr = err.ErrorOrNil()
	})
	return s.closeErr
}

func (s *Websocket[TIn, TOut]) runReadLoop(ctx context.Context, inbox chan<- TIn) error {
	s.log.Trace("running socket read loop")
	defer s.cancel()
	defer close(inbox)

	s.conn.SetReadLimit(maxMessageSize)
	if err := s.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return fmt.Errorf("setting initial read deadline: %w", err)
	}
	s.conn.SetPongHandler(func(string) error {
		if err := s.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			s.log.WithError(err).Error("setting read deadline")
		}
		return nil
	})

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

func (s *Websocket[TIn, TOut]) runWriteLoop(ctx context.Context, outbox <-chan TOut) error {
	s.log.Trace("running socket write loop")
	defer s.cancel()

	ping := time.NewTicker(pingInterval)
	defer ping.Stop()
	for {
		select {
		case msg := <-outbox:
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(msg); err != nil {
				return fmt.Errorf("encoding outbound message: %w", err)
			}

			if cur, max := buf.Len(), maxMessageSize; cur > max {
				return fmt.Errorf("message size %d exceeds maximum size %d", cur, max)
			}

			err := s.conn.WriteMessage(websocket.TextMessage, buf.Bytes())
			switch {
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

func (s *Websocket[TIn, TOut]) closeGraceful() error {
	s.cancel()

	closeDeadline := time.Now().Add(closeWait)
	s.conn.SetPongHandler(nil) // So the pong handler doesn't extend the deadline.
	if err := s.conn.SetReadDeadline(closeDeadline); err != nil {
		return fmt.Errorf("setting read deadline: %w", err)
	}

	// If this close message begins the handshake, the read loop will exhaust messages until our
	// peer responds with their close, or it exceeds the read deadline (or, you know, the underlying
	// connection is ripped from its hands), then exit. If we are already closed (have received and
	// responded with, by the default close handler, a close), we will receive ErrCloseSent from the
	// write and the read loop should have already exited.
	if err := s.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "close called"),
		closeDeadline,
	); err != websocket.ErrCloseSent && err != nil {
		return fmt.Errorf("sending close: %w", err)
	}

	<-s.Done
	if clErr := s.conn.Close(); clErr != nil {
		return fmt.Errorf("closing underlying conn: %w", clErr)
	}
	return nil
}

func (s *Websocket[TIn, TOut]) closeForced() error {
	s.cancel()
	if err := s.conn.Close(); err != nil {
		<-s.Done
		return fmt.Errorf("closing underlying conn: %w", err)
	}
	<-s.Done
	return nil
}

func (s *Websocket[TIn, TOut]) setError(err error) {
	s.errLock.Lock()
	defer s.errLock.Unlock()
	s.err = multierror.Append(s.err, err)
}
