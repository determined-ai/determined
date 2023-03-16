package dispatcherrm

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"golang.org/x/exp/slices"

	"github.com/determined-ai/determined/master/internal/db"
)

// dispatcherState is the Determined-persisted representation for any dispatcher state.
type dispatcherState struct {
	bun.BaseModel `bun:"table:resourcemanagers_dispatcher_rm_state"`
	*sync.RWMutex

	DisabledAgents []string `bun:"disabled_agents,array"`
}

func newDispatcherState() *dispatcherState {
	return &dispatcherState{RWMutex: &sync.RWMutex{}}
}

// getDispatcherState retrieves the current dispatcher state from the database.
func getDispatcherState(ctx context.Context) (*dispatcherState, error) {
	state := newDispatcherState()
	err := db.Bun().NewSelect().Model(state).Scan(ctx)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, fmt.Errorf("getting dispatcher state: %w", err)
	}
	return state, nil
}

// persist creates or updates the current dispatcher state in the database.
func (s *dispatcherState) persist(ctx context.Context) error {
	_, err := db.Bun().NewInsert().Model(s).On("conflict (id) do update").Exec(ctx)
	if err != nil {
		return fmt.Errorf("setting dispatcher state: %w", err)
	}
	return nil
}

// disableAgent adds the given agent to the list of disabled agents and persists the state.
func (s *dispatcherState) disableAgent(agentID string) error {
	s.Lock()
	defer s.Unlock()

	if slices.Index(s.DisabledAgents, agentID) != -1 {
		return errors.Errorf("agent %s already disabled", agentID)
	}

	s.DisabledAgents = append(s.DisabledAgents, agentID)

	if err := s.persist(context.TODO()); err != nil {
		return fmt.Errorf("agent %s disabled but may be enabled on server restart: %w", agentID, err)
	}
	return nil
}

// enableAgent removes the given agent from the list of disabled agents and persists the state.
func (s *dispatcherState) enableAgent(agentID string) error {
	s.Lock()
	defer s.Unlock()

	index := slices.Index(s.DisabledAgents, agentID)
	if index == -1 {
		return errors.Errorf("agent %s not disabled", agentID)
	}

	s.DisabledAgents = slices.Delete(s.DisabledAgents, index, index+1)

	if err := s.persist(context.TODO()); err != nil {
		return fmt.Errorf("agent %s enabled but may be disabled on server restart: %w", agentID, err)
	}
	return nil
}

// isAgentEnabled returns true if the given agent is not disabled.
func (s *dispatcherState) isAgentEnabled(agentID string) bool {
	s.RLock()
	defer s.RUnlock()

	return slices.Index(s.DisabledAgents, agentID) == -1
}
