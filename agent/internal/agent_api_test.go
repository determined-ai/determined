//go:build integration

package internal_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/agent/internal"
	"github.com/determined-ai/determined/agent/test/testutils"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/groupx/errgroupx"
	"github.com/determined-ai/determined/master/pkg/ws"
)

func TestAgentStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logC := testutils.NewLogChannel(1024)
	defer close(logC)

	log.AddHook(logC)
	log.SetLevel(log.TraceLevel)

	t.Log("running mock master")
	wg := errgroupx.WithContext(ctx)
	defer func() {
		_ = wg.Wait()
	}()
	defer wg.Cancel()

	t.Log("starting mock master")
	opts := testutils.DefaultAgentConfig(5)

	mockMasterHandler := wrap(t, func(c *websocket.Conn) error {
		socket := ws.Wrap[*aproto.MasterMessage, *aproto.AgentMessage]("server", c)
		defer func() {
			if err := socket.Close(); err != nil {
				t.Errorf("closing server websocket: %v", err)
				return
			}
		}()

		mopts := testutils.DefaultMasterSetAgentConfig()
		select {
		case socket.Outbox <- &aproto.AgentMessage{MasterSetAgentOptions: &mopts}:
		case <-ctx.Done():
			return nil
		}

		select {
		case msg := <-socket.Inbox:
			require.NotNil(t, msg.AgentStarted)
		case <-ctx.Done():
			return nil
		}

		<-ctx.Done()
		return nil
	})

	srv := http.Server{
		Addr:    fmt.Sprintf("localhost:%d", opts.MasterPort),
		Handler: mockMasterHandler,
	}
	wg.Go(func(ctx context.Context) error {
		return srv.ListenAndServe()
	})

	t.Log("looking for all clear")
	a := internal.New(ctx, "0.0.0-test", opts)
	for l := range logC {
		if strings.Contains(l.Message, "watching for ws requests and system events") {
			break
		}
	}

	t.Log("gracefully tearing down agent")
	cancel()
	require.NoError(t, a.Wait())
	require.NoError(t, srv.Shutdown(context.Background()))
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
