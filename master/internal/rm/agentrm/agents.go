package agentrm

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/connsave"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/queue"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

type agentUpdatedEvent struct {
	resourcePool string
}

// webSocketRequest notifies the actor that a websocket is attempting to connect.
type webSocketRequest struct {
	echoCtx echo.Context
}

// isReconnect checks if agent is reconnecting after a network failure.
func (w *webSocketRequest) isReconnect() (bool, error) {
	return strconv.ParseBool(w.echoCtx.QueryParam("reconnect"))
}

type agents struct {
	syslog *logrus.Entry
	mu     sync.Mutex

	agents       *tasklist.Registry[agentID, *agent]
	agentUpdates *queue.Queue[agentUpdatedEvent]
	poolConfigs  []config.ResourcePoolConfig
	opts         *aproto.MasterSetAgentOptions
}

func newAgentService(
	poolConfigs []config.ResourcePoolConfig,
	opts *aproto.MasterSetAgentOptions,
) (*agents, *queue.Queue[agentUpdatedEvent]) {
	agentUpdates := queue.New[agentUpdatedEvent]()
	a := &agents{
		syslog:       logrus.WithField("component", "agents"),
		agents:       tasklist.NewRegistry[agentID, *agent](),
		agentUpdates: agentUpdates,
		poolConfigs:  poolConfigs,
		opts:         opts,
	}

	// TODO(ilia): only restore the agents which have some non-zero state.
	// Currently, if an agent tries to reconnect and it was not restored here,
	// then it'd be told it must restart and do a fresh connection.
	agentStates, err := retrieveAgentStates()
	if err != nil {
		a.syslog.WithError(err).Warnf("failed to retrieve agent states")
	}

	a.syslog.Debugf("agent states to restore: %d", len(agentStates))
	badAgentIds := []agentID{}

	for agentID, state := range agentStates {
		state := state
		agentRef, err := a.createAgent(agentID, state.resourcePoolName, a.opts, &state, func() {
			_ = a.agents.Delete(agentID)
		})
		if err != nil {
			a.syslog.WithError(err).Warnf("failed to create agent %s", agentID)
			badAgentIds = append(badAgentIds, agentID)
			continue
		}

		err = a.agents.Add(agentID, agentRef)
		if err != nil {
			a.syslog.WithError(err).Warnf("tried to restore duplicate agent %s", agentID)
			badAgentIds = append(badAgentIds, agentID)
			continue
		}

		a.syslog.Debugf("restored agent state: %s", agentID)
	}

	if len(badAgentIds) > 0 {
		a.syslog.Debugf("cleaning %d bad agent states", len(badAgentIds))
		if err := clearAgentStates(badAgentIds); err != nil {
			a.syslog.WithError(err).Warnf("failed to clean bad agent states")
		}
	}

	return a, agentUpdates
}

// list implements agentService.
func (a *agents) list(resourcePoolName string) map[agentID]*agentState {
	agents := a.agents.Snapshot()
	result := make(map[agentID]*agentState, len(agents))
	for id, a := range agents {
		state, err := a.State()
		if err != nil {
			a.syslog.WithError(err).Warnf("failed to get agent state for agent %s", id)
			continue
		}
		if state.resourcePoolName != resourcePoolName {
			continue
		}
		result[state.id] = state
	}
	return result
}

func (a *agents) get(id agentID) (*agent, bool) {
	return a.agents.Load(id)
}

func (a *agents) HandleWebsocketConnection(msg webSocketRequest) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	cmuxConn := connsave.GetConn(msg.echoCtx.Request().Context()).(*cmux.MuxConn)
	// Here, we just have to check that there are any certificates at all, since the top-level TLS
	// config verifies that any certificates that are provided are valid.
	if tlsConn, ok := cmuxConn.Conn.(*tls.Conn); ok {
		r, ok := config.GetMasterConfig().GetAgentRMConfig()
		if !ok {
			return fmt.Errorf("can't read agent config")
		}
		requireAuth := r.ResourceManager.AgentRM.RequireAuthentication

		missingAuth := len(tlsConn.ConnectionState().PeerCertificates) == 0
		if requireAuth && missingAuth {
			a.syslog.WithField("remote-addr", tlsConn.RemoteAddr()).
				Warnf("rejecting agent WebSocket request with no certificates")
			return echo.ErrForbidden
		}
	}

	id := msg.echoCtx.QueryParam("id")
	reconnect, err := msg.isReconnect()
	if err != nil {
		return errors.Wrapf(err, "parsing reconnect query param")
	}

	// If the agent actor is still alive on our side when an agent tries to reconnect,
	// accept it. Whether it is a network failure or a crash/restart, we will just try
	// to reattach whatever containers still exist.
	// That logic is located in agent.receive(ws.WebSocketRequest).
	existingRef, ok := a.agents.Load(agentID(id))
	if ok {
		a.syslog.WithField("reconnect", reconnect).Infof("restoring agent id: %s", id)
		existingRef.HandleWebsocketConnection(msg)
		return nil
	}

	// If the agent actor is _not_ alive on our side and the agent is trying to reconnect,
	// continue to deny it. This case is nearly impossible (master waits longer than agent
	// tries, to avoid it).
	if reconnect {
		return aproto.ErrAgentMustReconnect
	}

	// Finally, this must not be a recovery flow, so just create the agent actor.
	resourcePool := msg.echoCtx.QueryParam("resource_pool")
	ref, err := a.createAgent(agentID(id), resourcePool, a.opts, nil, func() { _ = a.agents.Delete(agentID(id)) })
	if err != nil {
		return err
	}

	err = a.agents.Add(agentID(id), ref)
	if err != nil {
		return fmt.Errorf("adding agent because of incoming websocket: %w", err)
	}
	ref.HandleWebsocketConnection(msg)
	return nil
}

func (a *agents) getAgents() *apiv1.GetAgentsResponse {
	var response apiv1.GetAgentsResponse
	for _, a := range a.summarize() {
		response.Agents = append(response.Agents, a.ToProto())
	}
	return &response
}

func (a *agents) createAgent(
	id agentID,
	resourcePool string,
	opts *aproto.MasterSetAgentOptions,
	restoredAgentState *agentState,
	unregister func(),
) (*agent, error) {
	if id == "" {
		return nil, errors.Errorf("invalid agent id specified: %s", id)
	}
	if resourcePool == "" {
		a.syslog.Info("resource pool is empty; using default resource pool: default")
		resourcePool = "default"
	}

	var poolConfig *config.ResourcePoolConfig
	for _, pc := range a.poolConfigs {
		pc := pc
		if pc.PoolName == resourcePool {
			poolConfig = &pc
			break
		}
	}
	if poolConfig == nil {
		return nil, fmt.Errorf("cannot find specified resource pool %s for agent %s", resourcePool, id)
	}

	return newAgent(
		id,
		a.agentUpdates,
		resourcePool,
		poolConfig,
		opts,
		restoredAgentState,
		unregister,
	), nil
}

func (a *agents) summarize() model.AgentsSummary {
	agents := a.agents.Snapshot()
	summary := make(map[string]model.AgentSummary, len(agents))
	for id, a := range agents {
		summary[string(id)] = a.Summarize()
	}
	return summary
}
