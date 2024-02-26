//go:build integration
// +build integration

package rm

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/ws"
	"github.com/determined-ai/determined/master/test/testutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"gotest.tools/assert"
)

func TestAgentConnection(t *testing.T) {
	testAgentID := "test-agent0"

	masterCfg, err := testutils.DefaultMasterConfig()
	assert.NilError(t, err, "failed to obtain master config")
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	t.Log("starting first agent, which should be healthy")
	_, err = connectFakeAgent(creds, t, masterCfg.Port, testAgentID)
	assert.NilError(t, err, "error connecting first agent")
	apiResp, err := cl.GetAgent(creds, &apiv1.GetAgentRequest{
		AgentId: testAgentID,
	})
	assert.NilError(t, err, "could not connect to master api to get agent")
	assert.Assert(t, apiResp.GetAgent().GetEnabled())
	assert.Assert(t, !apiResp.GetAgent().GetDraining())

	t.Log("starting second agent, which should not stop the first agent")
	_, err = connectFakeAgent(creds, t, masterCfg.Port, testAgentID)
	assert.ErrorContains(t, err, "websocket already connected")
	apiResp, err = cl.GetAgent(creds, &apiv1.GetAgentRequest{
		AgentId: testAgentID,
	})

	t.Log("checking to make sure an agent is still enabled")
	assert.NilError(t, err, "could not connect to master api to get agent")
	assert.Assert(t, apiResp.Agent.GetEnabled())
	assert.Assert(t, !apiResp.Agent.GetDraining())
}

func connectFakeAgent(
	ctx context.Context,
	t *testing.T,
	port int,
	agentID string,
) (*ws.WebSocket[*aproto.AgentMessage, *aproto.MasterMessage], error) {
	dialer := websocket.Dialer{
		Proxy:            websocket.DefaultDialer.Proxy,
		HandshakeTimeout: websocket.DefaultDialer.HandshakeTimeout,
	}

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("unable to get hostname: %s", err)
	}

	masterAddr := fmt.Sprintf(
		"ws://localhost:%d/agents?id=%s&version=%s&reconnect=false&hostname=%s",
		port, agentID, "dev", hostname,
	)
	t.Logf("connecting mock agent to master at: %s", masterAddr)
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
