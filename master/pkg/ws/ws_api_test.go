package ws_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/ws"
)

func TestWebsocket(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Log("running server")
	s := httptest.NewServer(wrap(t, func(c *websocket.Conn) error {
		ws, err := ws.Wrap[int, int]("server", c)
		if err != nil {
			return err
		}
		defer func() {
			if err := ws.Close(); err != nil {
				t.Errorf("closing websocket: %v", err)
				return
			}
		}()

		t.Log("server read")
		var x int
		select {
		case tmp, ok := <-ws.Inbox:
			if !ok {
				return fmt.Errorf("server read (inbox closed early): %w", ws.Error())
			}
			x = tmp
		case <-ctx.Done():
			return nil
		}

		t.Log("server write")
		select {
		case ws.Outbox <- x + 1:
		case <-ctx.Done():
			return nil
		case <-ws.Done:
			return fmt.Errorf("server write (closed early): %w", ws.Error())
		}

		<-ctx.Done()
		return nil
	}))
	defer s.Close()

	t.Log("connecting websocket to server")
	c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http"), nil)
	if err != nil {
		t.Errorf("failed to dial websocket: %v", err)
		return
	}
	ws, err := ws.Wrap[int, int]("client", c)
	if err != nil {
		t.Errorf("failed to wrap websocket: %v", err)
		return
	}
	defer func() {
		if err := ws.Close(); err != nil {
			t.Errorf("closing the websocket: %s", err)
			return
		}
	}()

	t.Log("client write")
	x := 0
	select {
	case ws.Outbox <- x:
	case <-ws.Done:
		t.Errorf("client write (closed early): %s", ws.Error())
		return
	}

	t.Log("client read")
	y, ok := <-ws.Inbox
	if !ok {
		t.Errorf("client read (inbox closed early): %s", ws.Error())
		return
	}
	require.Equal(t, x+1, y)
}

func TestWebsocketConcurrent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type testMessage struct {
		ID  uuid.UUID
		Num int
	}

	t.Log("running server")
	s := httptest.NewServer(wrap(t, func(c *websocket.Conn) error {
		ws, err := ws.Wrap[testMessage, testMessage]("server", c)
		if err != nil {
			return err
		}
		defer func() {
			if err := ws.Close(); err != nil {
				t.Errorf("closing server websocket: %v", err)
				return
			}
		}()

		for {
			var msg testMessage
			select {
			case tmp, ok := <-ws.Inbox:
				if !ok {
					return fmt.Errorf("server read (inbox closed early): %w", ws.Error())
				}
				msg = tmp
			case <-ctx.Done():
				return nil
			}
			t.Logf("server read: %s %d", msg.ID, msg.Num)

			select {
			case ws.Outbox <- testMessage{ID: msg.ID, Num: msg.Num + 1}:
			case <-ws.Done:
				return fmt.Errorf("server write {%s, %d}: %w", msg.ID, msg.Num, ws.Error())
			case <-ctx.Done():
				return nil
			}
			t.Logf("server write: %v", msg)
		}
	}))
	defer s.Close()

	t.Log("connecting to websocket server")
	c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http"), nil)
	if err != nil {
		t.Errorf("dialing websocket: %v", err)
		return
	}
	ws, err := ws.Wrap[testMessage, testMessage]("client", c)
	if err != nil {
		t.Errorf("wrapping websocket: %v", err)
		return
	}
	defer func() {
		if err := ws.Close(); err != nil {
			t.Errorf("closing the websocket: %s", err)
		}
	}()

	t.Log("spinning up workers, doing incrementing back and forth")
	const (
		workersCount = 10
		iterations   = 100
	)
	var workerWg sync.WaitGroup
	workers := make(map[uuid.UUID]chan<- int, workersCount)
	for i := 0; i < workersCount; i++ {
		workerID := uuid.New()
		workerChan := make(chan int, 1)
		workers[workerID] = workerChan
		workerWg.Add(1)
		go func(id uuid.UUID, c <-chan int) {
			defer workerWg.Done()
			for i := 0; i < iterations; i++ {
				select {
				case ws.Outbox <- testMessage{ID: id, Num: i}:
				case <-ws.Done:
					t.Errorf("worker %s write %d: %s", id, i, ws.Error())
					return
				}
				t.Logf("worker %s wrote %d", id, i)

				select {
				case msg := <-c:
					t.Logf("worker %s read %d", id, msg)
					require.Equal(t, i+1, msg, "unexpected response for worker: %v", id)
				case <-ctx.Done():
					return
				}
			}
		}(workerID, workerChan)
	}

	t.Log("running hub, to direct messages from the inbox down to workers")
	go func() {
		for {
			select {
			case msg, ok := <-ws.Inbox:
				switch {
				case ctx.Err() != nil:
					return
				case !ok:
					t.Errorf("client inbox closed prematurely: %v", ws.Error())
					return
				}
				t.Logf("hub read %d for %s", msg.Num, msg.ID)
				workers[msg.ID] <- msg.Num
			case <-ws.Done:
				switch {
				case ctx.Err() != nil:
					return
				case ws.Error() != nil:
					t.Errorf("client ws failed: %v", ws.Error())
					return
				default:
					t.Errorf("client ws closed prematurely: %v", ws.Error())
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	t.Log("waiting on workers")
	workerWg.Wait()

	t.Log("cleaning up")
	cancel()
}

func wrap(t *testing.T, handler func(*websocket.Conn) error) http.HandlerFunc {
	upgrader := websocket.Upgrader{}
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade to websocket failed: %s", err)
			return
		}

		if err := handler(c); err != nil {
			t.Errorf("websocket failed: %s", err)
			return
		}
	}
}
