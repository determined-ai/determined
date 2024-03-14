package agentrm

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/queue"
	"github.com/determined-ai/determined/master/pkg/ws"
)

func TestAgentFastFailAfterFirstConnect2(t *testing.T) {
	var closed atomic.Bool
	a := newAgent(
		"test",
		queue.New[agentUpdatedEvent](),
		"default",
		&config.ResourcePoolConfig{},
		&aproto.MasterSetAgentOptions{
			MasterInfo: aproto.MasterInfo{},
			LoggingOptions: model.LoggingConfig{
				DefaultLoggingConfig: &model.DefaultLoggingConfig{},
			},
			ContainersToReattach: []aproto.ContainerReattach{},
		},
		nil,
		func() { closed.Store(true) },
	)

	// Connect a fake websocket.
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		err := a.HandleWebsocketConnection(webSocketRequest{echoCtx: c})
		require.NoError(t, err)
		return nil
	})
	server := httptest.NewServer(e.Server.Handler)

	var dialer websocket.Dialer
	conn, _, err := dialer.Dial(fmt.Sprintf("ws://%s", strings.TrimPrefix(server.URL, "http://")), nil)
	require.NoError(t, err)
	_, err = ws.Wrap[*aproto.MasterMessage, aproto.AgentMessage]("test", conn)
	require.NoError(t, err)

	// Close the underlying conn to simulate a failure.
	err = conn.UnderlyingConn().Close()
	require.NoError(t, err)

	for {
		if closed.Load() {
			// The agent should close without a panic. A panic in the agent would bubble up and fail this test.
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
