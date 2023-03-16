//go:build integration
// +build integration

package dispatcherrm

import (
	"context"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/etc"

	"gotest.tools/assert"
)

func TestDispatcherAgents(t *testing.T) {
	assert.NilError(t, etc.SetRootPath(db.RootFromDB))
	_ = db.MustResolveTestPostgres(t)

	// clear any existing state
	_, _ = db.Bun().NewDelete().Model(&dispatcherState{}).Where("id=0").Exec(context.TODO())

	m := &dispatcherResourceManager{
		resourceDetails: hpcResourceDetailsCache{
			lastSample: hpcResources{
				Nodes: []hpcNodeDetails{
					{Name: "Node A"},
					{Name: "Node B"},
					{Name: "Node C"},
				},
			},
			sampleTime: time.Now(),
		},
		dbState: *newDispatcherState(),
	}
	_, err := m.disableAgent("Node Z")
	assert.Check(t, err != nil)

	ctx := &actor.Context{}
	resp := m.generateGetAgentsResponse(ctx)
	assert.Equal(t, len(resp.Agents), 3)
	for i := range resp.Agents {
		assert.Equal(t, resp.Agents[i].Enabled, true)
	}

	_, err = m.disableAgent("Node A")
	assert.NilError(t, err)
	_, err = m.disableAgent("Node B")
	assert.NilError(t, err)
	_, err = m.disableAgent("Node C")
	assert.NilError(t, err)

	resp = m.generateGetAgentsResponse(ctx)
	assert.Equal(t, len(resp.Agents), 3)
	for i := range resp.Agents {
		assert.Equal(t, resp.Agents[i].Enabled, false)
	}

	_, err = m.enableAgent("Node A")
	assert.NilError(t, err)
	_, err = m.enableAgent("Node C")
	assert.NilError(t, err)

	resp = m.generateGetAgentsResponse(ctx)
	assert.Equal(t, len(resp.Agents), 3)
	for i := range resp.Agents {
		assert.Equal(t, resp.Agents[i].Enabled, resp.Agents[i].Id != "Node B")
	}
}
