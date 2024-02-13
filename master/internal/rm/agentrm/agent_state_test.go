//go:build integration

package agentrm

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/taskmodel"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func TestMain(m *testing.M) {
	pgDB, err := db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	os.Exit(m.Run())
}

func TestAgentStatePersistence(t *testing.T) {
	// Clear all agent states.
	_, err := db.Bun().NewDelete().Model((*agentSnapshot)(nil)).Where("1 = 1").Exec(context.TODO())
	require.NoError(t, err)

	// Fake an agent, test adding it to the db.
	state := newAgentState(agentID(uuid.NewString()), 64)
	state.handler = &agent{}
	state.resourcePoolName = "compute"
	devices := []device.Device{
		{
			ID:    0,
			Brand: "nvda",
			UUID:  uuid.NewString(),
			Type:  "3090",
		},
		{
			ID:    1,
			Brand: "nvda",
			UUID:  uuid.NewString(),
			Type:  "3090",
		},
	}
	started := &aproto.AgentStarted{
		Version:              "",
		Devices:              devices,
		ContainersReattached: []aproto.ContainerReattachAck{},
	}
	state.agentStarted(started)
	require.Equal(t, 2, len(state.getSlotsSummary("/myagent")))

	// Run through some container states.
	tID := model.TaskID(uuid.NewString())
	err = db.AddTask(context.TODO(), &model.Task{
		TaskID:     tID,
		JobID:      nil,
		TaskType:   model.TaskTypeCommand,
		StartTime:  time.Now(),
		LogVersion: model.CurrentTaskLogVersion,
	})
	require.NoError(t, err)

	aID := model.AllocationID(uuid.NewString())
	err = db.AddAllocation(context.TODO(), &model.Allocation{
		AllocationID: aID,
		TaskID:       tID,
		Slots:        2,
		ResourcePool: "compute",
		StartTime:    ptrs.Ptr(time.Now()),
		State:        ptrs.Ptr(model.AllocationStateAssigned),
		Ports:        map[string]int{"ok": 8888},
	})
	require.NoError(t, err)

	cID := cproto.ID(uuid.NewString())
	container := cproto.Container{
		ID:          cID,
		State:       cproto.Assigned,
		Devices:     devices,
		Description: "some job",
	}
	err = state.startContainer(sproto.StartTaskContainer{
		AllocationID: aID,
		StartContainer: aproto.StartContainer{
			Container: container,
			Spec:      cproto.Spec{},
		},
	})
	require.NoError(t, err)

	_, err = db.Bun().NewInsert().Model(&taskmodel.ResourcesWithState{
		ResourceID:   sproto.ResourcesID(cID),
		AllocationID: aID,
		Container:    &container,
	}).Exec(context.Background())
	require.NoError(t, err)

	_, err = db.Bun().NewInsert().Model(&containerSnapshot{
		ResourceID: sproto.ResourcesID(cID),
		AgentID:    "myagent",
		ID:         cID,
	}).Exec(context.TODO())
	require.NoError(t, err)

	state.containerStateChanged(aproto.ContainerStateChanged{
		Container: cproto.Container{
			ID:          cID,
			State:       cproto.Running,
			Devices:     devices,
			Description: "some job",
		},
		ContainerStarted: &aproto.ContainerStarted{
			ProxyAddress: "localhost",
		},
	})
	require.Equal(t, cproto.Running, state.containerState[cID].State)

	// Ensure agent state is retrievable and looks right, for crashes.
	states, err := retrieveAgentStates()
	require.NoError(t, err)
	require.Len(t, states, 1)
	var restored *agentState
	for _, s := range states {
		s := s
		restored = &s
		break
	}
	require.NotNil(t, restored)
	require.Equal(t, state.id, restored.id)

	// And test restoring the existence of containers.
	err = restored.restoreContainersField()
	require.NoError(t, err)
	require.Equal(t, 1, len(restored.containerAllocation))

	// And it is correctly kept if it is recovered.
	err = state.clearUnlessRecovered(map[cproto.ID]aproto.ContainerReattachAck{
		cID: {Container: container},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(restored.containerAllocation))

	// Containers removed after exit.
	state.containerStateChanged(aproto.ContainerStateChanged{
		Container: cproto.Container{
			ID:          cID,
			State:       cproto.Terminated,
			Devices:     devices,
			Description: "some job",
		},
		ContainerStopped: &aproto.ContainerStopped{},
	})
	require.NotContains(t, state.containerState, cID)

	// Test deleting the state.
	err = state.delete()
	require.NoError(t, err)
	exists, err := db.Bun().NewSelect().Model((*agentSnapshot)(nil)).Where("agent_id = ?", state.id).Exists(context.TODO())
	require.NoError(t, err)
	require.False(t, exists)
}

func TestClearAgentStates(t *testing.T) {
	ctx := context.Background()
	agentIDs := []agentID{agentID(uuid.NewString()), agentID(uuid.NewString())}
	for _, agentID := range agentIDs {
		_, err := db.Bun().NewInsert().Model(&agentSnapshot{
			AgentID:               agentID,
			UUID:                  uuid.NewString(),
			ResourcePoolName:      "rp-name",
			Label:                 "label",
			MaxZeroSlotContainers: 0,
		}).Exec(ctx)
		require.NoError(t, err)
	}

	require.NoError(t, clearAgentStates(agentIDs))
	exists, err := db.Bun().NewSelect().Model(&agentSnapshot{}).
		Where("agent_id IN (?)", bun.In(agentIDs)).
		Exists(ctx)
	require.NoError(t, err)
	require.False(t, exists)
}

func Test_agentState_checkAgentStartedDevicesMatch(t *testing.T) {
	stableUUID := uuid.NewString()
	tests := []struct {
		name            string
		state           agentState
		agentStarted    *aproto.AgentStarted
		wantErrContains string
	}{
		{
			name: "devices match",
			state: agentState{
				slotStates: map[device.ID]*slot{
					0: {
						device: device.Device{
							ID:    0,
							Brand: "nvda",
							UUID:  stableUUID,
							Type:  "3090",
						},
					},
				},
			},
			agentStarted: &aproto.AgentStarted{Devices: []device.Device{
				{
					ID:    0,
					Brand: "nvda",
					UUID:  stableUUID,
					Type:  "3090",
				},
			}},
			wantErrContains: "",
		},
		{
			name: "device is missing",
			state: agentState{
				slotStates: map[device.ID]*slot{
					0: {
						device: device.Device{
							ID:    0,
							Brand: "nvda",
							UUID:  uuid.NewString(),
							Type:  "3090",
						},
					},
				},
			},
			agentStarted:    &aproto.AgentStarted{Devices: nil},
			wantErrContains: "device count has changed",
		},
		{
			name: "extra device",
			state: agentState{
				slotStates: map[device.ID]*slot{
					0: {
						device: device.Device{
							ID:    0,
							Brand: "nvda",
							UUID:  uuid.NewString(),
							Type:  "3090",
						},
					},
				},
			},
			agentStarted: &aproto.AgentStarted{Devices: []device.Device{
				{
					ID:    0,
					Brand: "nvda",
					UUID:  uuid.NewString(),
					Type:  "3090",
				},
				{
					ID:    1,
					Brand: "nvda",
					UUID:  uuid.NewString(),
					Type:  "3090",
				},
			}},
			wantErrContains: "device count has changed",
		},
		{
			name: "mismatched devices device",
			state: agentState{
				slotStates: map[device.ID]*slot{
					0: {
						device: device.Device{
							ID:    0,
							Brand: "nvda",
							UUID:  uuid.NewString(),
							Type:  "3090",
						},
					},
				},
			},
			agentStarted: &aproto.AgentStarted{Devices: []device.Device{
				{
					ID:    0,
					Brand: "nvda",
					UUID:  uuid.NewString(),
					Type:  "4090",
				},
			}},
			wantErrContains: "device properties have changed",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.checkAgentStartedDevicesMatch(tt.agentStarted)
			if tt.wantErrContains == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErrContains)
		})
	}
}

func TestSlotStates(t *testing.T) {
	rpName := "test"
	state := newAgentState(agentID(uuid.NewString()), 64)
	state.handler = &agent{}
	state.resourcePoolName = rpName
	devices := []device.Device{
		{
			ID:    0,
			Brand: "nvda",
			UUID:  uuid.NewString(),
			Type:  "3090",
		},
		{
			ID:    1,
			Brand: "nvda",
			UUID:  uuid.NewString(),
			Type:  "3090",
		},
	}
	started := &aproto.AgentStarted{
		Version:              "",
		Devices:              devices,
		ContainersReattached: []aproto.ContainerReattachAck{},
	}
	state.agentStarted(started)
	slots := state.getSlotsSummary("/")
	require.Equal(t, 2, state.numSlots())
	for _, s := range slots {
		require.True(t, s.Enabled)
		require.False(t, s.Draining)
	}

	slot, err := state.patchSlotState(patchSlotState{
		id:      0,
		enabled: ptrs.Ptr(false),
		drain:   ptrs.Ptr(true),
	})
	require.NoError(t, err)
	require.Equal(t, true, slot.Draining)
	require.Equal(t, false, slot.Enabled)
	require.Equal(t, 2, state.numSlots())

	slots = state.patchAllSlotsState(patchAllSlotsState{
		enabled: ptrs.Ptr(true),
	})
	require.Equal(t, 2, len(slots))
	for _, s := range slots {
		require.True(t, s.Enabled)
	}

	// Manipulate agent states a bit and check slot counts.
	state.Devices[devices[0]] = ptrs.Ptr(cproto.NewID())
	state.disable(true)
	require.Equal(t, 1, state.numSlots())

	state.Devices[devices[0]] = nil
	require.Equal(t, 0, state.numSlots())

	state.disable(false)
	require.Equal(t, 0, state.numSlots())

	state.enable()
	require.Equal(t, 2, state.numSlots())
}
