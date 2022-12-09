package internal

import (
	"net"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type systemForRWCoordinator struct {
	system *actor.System
	t      *testing.T
}

var upgrader = websocket.Upgrader{}

func (s *systemForRWCoordinator) requestHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.t.Errorf("Error: %s", err)
	}

	resourceName := r.URL.Path
	query := r.URL.Query()

	readLockString := query["read_lock"]
	var readLock bool
	if readLockString[0] == "False" {
		readLock = false
	} else {
		readLock = true
	}

	socketActor := s.system.AskAt(actor.Addr("rwCoordinator"),
		resourceRequest{resourceName, readLock, conn})
	actorRef := socketActor.Get().(*actor.Ref)

	// Wait for the websocket actor to terminate.
	if err := actorRef.AwaitTermination(); err != nil {
		s.t.Logf("Server socket closed")
	}
}

func readValue(t *testing.T, addr string, sleepTime time.Duration, wg *sync.WaitGroup) {
	defer wg.Done()
	u := url.URL{
		Scheme:   "ws",
		Host:     addr,
		Path:     "/ws/data-layer/resource1",
		RawQuery: "read_lock=True",
	}
	c, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	assert.NilError(t, err)
	defer func() {
		resp.Close = true
		if errClose := c.Close(); errClose != nil {
			t.Logf("Error closing socket: %s", errClose)
		}
	}()

	_, message, err := c.ReadMessage()
	assert.NilError(t, err)
	assert.Equal(t, string(message),
		"read_lock_granted", "Did not receive `read_lock_granted` "+
			"response from server, got instead: %s", string(message))

	time.Sleep(sleepTime)
}

func writeValue(
	t *testing.T,
	addr string,
	sleepTime time.Duration,
	wg *sync.WaitGroup,
	sharedValue *int,
) {
	defer wg.Done()
	u := url.URL{
		Scheme:   "ws",
		Host:     addr,
		Path:     "/ws/data-layer/resource1",
		RawQuery: "read_lock=False",
	}
	c, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	assert.NilError(t, err)
	defer func() {
		resp.Close = true
		if errClose := c.Close(); errClose != nil {
			t.Logf("Error closing socket: %s", errClose)
		}
	}()

	_, message, err := c.ReadMessage()
	assert.NilError(t, err)
	assert.Equal(t, string(message),
		"write_lock_granted", "Did not receive `write_lock_granted` "+
			"response from server: %s", string(message))

	time.Sleep(sleepTime)
	*sharedValue++
}

func TestRWCoordinatorLayer(t *testing.T) {
	numThreads := 2
	sharedValue := 0
	var wg sync.WaitGroup

	system := actor.NewSystem("")
	rwCoordinator := newRWCoordinator()
	_, created := system.ActorOf(actor.Addr("rwCoordinator"), rwCoordinator)
	if !created {
		t.Fatal("unable to create RW coordinator")
	}
	systemRef := &systemForRWCoordinator{system, t}

	// Start listening on a random port.
	l, err := net.Listen("tcp", "localhost:0")
	assert.NilError(t, err)
	defer func() {
		_ = l.Close()
	}()

	serverMux := http.NewServeMux()
	server := http.Server{Addr: l.Addr().String(), Handler: serverMux}
	serverMux.HandleFunc("/ws/data-layer/", systemRef.requestHandler)

	srverr := make(chan error)
	var closed bool
	defer func() {
		if !closed {
			_ = server.Close()
		}
	}()

	go func() {
		defer close(srverr)
		err := server.Serve(l)
		switch err {
		case http.ErrServerClosed:
			// We expect this when we close it.
		case nil:
			srverr <- errors.New("expected http.ErrServerClosed error but got nil")
		default:
			srverr <- err
		}
	}()

	wg.Add(numThreads * 2)
	for i := 0; i < numThreads; i++ {
		go readValue(t, l.Addr().String(), time.Duration(i)*time.Second, &wg)
		go writeValue(t, l.Addr().String(), time.Duration(i)*time.Second, &wg, &sharedValue)
	}
	wg.Wait()
	assert.Equal(t, sharedValue, numThreads)

	err = server.Close()
	closed = true
	assert.NilError(t, err)

	// Wait for the server to exit.
	assert.NilError(t, <-srverr)
}
