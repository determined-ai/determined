//go:build integration

package agentrm_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/ws"
	"github.com/determined-ai/determined/master/test/testutils/fixtures"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestAgentsDuplicateConnectionIdHandling(t *testing.T) {
	testAgentID := "test-agent0"

	masterCfg, err := fixtures.DefaultMasterConfig()
	require.NoError(t, err, "failed to obtain master config")
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := fixtures.RunMaster(ctx, nil)
	defer cancel()
	require.NoError(t, err, "failed to start master")

	t.Log("starting first agent, which should be healthy")
	_, err = connectFakeAgent(creds, t, masterCfg.Port, testAgentID)
	require.NoError(t, err, "error connecting first agent")
	apiResp, err := cl.GetAgent(creds, &apiv1.GetAgentRequest{
		AgentId: testAgentID,
	})
	require.NoError(t, err, "could not connect to master api to get agent")
	require.True(t, apiResp.GetAgent().GetEnabled())
	require.False(t, apiResp.GetAgent().GetDraining())

	t.Log("starting second agent, which should not stop the first agent")
	_, err = connectFakeAgent(creds, t, masterCfg.Port, testAgentID)
	require.ErrorContains(t, err, "websocket already connected")
	apiResp, err = cl.GetAgent(creds, &apiv1.GetAgentRequest{
		AgentId: testAgentID,
	})

	t.Log("checking to make sure an agent is still enabled")
	require.NoError(t, err, "could not connect to master api to get agent")
	require.True(t, apiResp.GetAgent().GetEnabled())
	require.False(t, apiResp.GetAgent().GetDraining())
}

func connectFakeAgent(
	ctx context.Context,
	t *testing.T,
	port int,
	agentID string,
) (*ws.WebSocket[*aproto.AgentMessage, *aproto.MasterMessage], error) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("unable to get hostname: %s", err)
	}
	masterAddr := fmt.Sprintf(
		"ws://localhost:%d/agents?id=%s&version=dev&reconnect=false&hostname=%s",
		port, agentID, hostname,
	)

	t.Logf("connecting mock agent to master at: %s", masterAddr)
	dialer := websocket.Dialer{
		Proxy:            websocket.DefaultDialer.Proxy,
		HandshakeTimeout: websocket.DefaultDialer.HandshakeTimeout,
	}
	conn, resp, err := dialer.DialContext(ctx, masterAddr, nil)
	if resp != nil {
		defer func() {
			if err = resp.Body.Close(); err != nil {
				t.Errorf("failed to close master response on connection: %s", err)
			}
		}()
	}
	if err != nil {
		if resp == nil {
			return nil, errors.Wrap(err, "error dialing master")
		}

		b, rErr := io.ReadAll(resp.Body)
		if rErr == nil && strings.Contains(string(b), aproto.ErrAgentMustReconnect.Error()) {
			return nil, aproto.ErrAgentMustReconnect
		}

		return nil, errors.Wrapf(err, "error dialing master: %s", b)
	}
	return ws.Wrap[*aproto.AgentMessage, *aproto.MasterMessage]("agent-"+agentID, conn)
}
