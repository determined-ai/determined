//go:build integration
// +build integration

package dispatcherrm

import (
	"context"
	"reflect"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"

	"gotest.tools/assert"
)

func TestDispatcherStatePersistence(t *testing.T) {
	assert.NilError(t, etc.SetRootPath(db.RootFromDB))
	_ = db.MustResolveTestPostgres(t)

	// clear any existing state
	_, _ = db.Bun().NewDelete().Model(&dispatcherState{}).Where("id=0").Exec(context.TODO())

	state, err := getDispatcherState(context.TODO())
	// expect empty state to exist initially
	assert.NilError(t, err)
	assert.Equal(t, len(state.DisabledAgents), 0)

	state.DisabledAgents = []string{"agent1", "agent2"}

	assert.NilError(t, state.persist(context.TODO()))

	state, err = getDispatcherState(context.TODO())
	assert.NilError(t, err)
	assert.Check(t, reflect.DeepEqual(state.DisabledAgents, []string{"agent1", "agent2"}))

	assert.Check(t, !state.isAgentEnabled("agent1"))
	assert.Check(t, state.isAgentEnabled("agentUnknown"))

	assert.ErrorContains(t, state.disableAgent("agent1"), "already disabled")
	assert.Check(t, !state.isAgentEnabled("agent1"))

	assert.NilError(t, state.enableAgent("agent1"))
	assert.Check(t, state.isAgentEnabled("agent1"))

	assert.NilError(t, state.disableAgent("agent1"))
	assert.Check(t, !state.isAgentEnabled("agent1"))

	assert.NilError(t, state.enableAgent("agent1"))

	assert.NilError(t, state.persist(context.TODO()))

	state, err = getDispatcherState(context.TODO())
	assert.NilError(t, err)
	assert.Check(t, reflect.DeepEqual(state.DisabledAgents, []string{"agent2"}))
}
