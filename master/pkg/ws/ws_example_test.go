package ws_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/determined-ai/determined/master/pkg/ws"
)

func Example() {
	// Start a Websocket server that converts ints to strings.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		// Send and receive messages from the client.
		s, err := ws.Wrap[int, string]("int2str", c)
		if err != nil {
			log.Println(err)
			return
		}
		for {
			num, ok := <-s.Inbox
			if !ok {
				log.Println(s.Error())
				return
			}

			select {
			case s.Outbox <- strconv.Itoa(num):
			case <-s.Done:
				log.Println(s.Error())
				return
			}
		}
	}))
	defer ts.Close()

	// Connect a Websocket to the server with using `github.com/gorilla/websocket`.
	c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http"), nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Use Wrap to wrap the underlying connection.
	s, err := ws.Wrap[string, int]("client", c)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		// Gracefully close our connection, using Close. If it fails, it will forcibly clean up.
		if err := s.Close(); err != nil {
			log.Println(err)
			return
		}
	}()

	// Send and receive messages to the server.
	select {
	case s.Outbox <- 42:
	case <-s.Done:
		log.Println(s.Error())
		return
	}

	str, ok := <-s.Inbox
	if !ok {
		log.Println(s.Error())
		return
	}
	fmt.Println(str)
	// Output: 42
}
