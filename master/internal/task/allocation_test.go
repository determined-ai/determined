package task

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/device"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

func TestAllocation(t *testing.T) {
	cases := []struct {
		name  string
		err   *sproto.ResourcesFailure
		acked bool
		exit  *AllocationExited
	}{
		{
			name:  "happy path",
			acked: true,
			exit:  &AllocationExited{},
		},
		{
			name:  "user requested stop",
			acked: false,
			exit:  &AllocationExited{UserRequestedStop: true},
		},
		{
			name:  "container failed",
			acked: false,
			err:   &sproto.ResourcesFailure{FailureType: sproto.ContainerFailed},
			exit:  &AllocationExited{Err: sproto.ResourcesFailure{FailureType: sproto.ContainerFailed}},
		},
		{
			name:  "container failed, but acked preemption",
			acked: true,
			err:   &sproto.ResourcesFailure{FailureType: sproto.ContainerFailed},
			exit:  &AllocationExited{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			system, _, rm, trialImpl, trial, db, a, self := setup(t)

			// Pre-allocated stage.
			mockRsvn := func(rID sproto.ResourcesID, agentID string) sproto.Resources {
				rsrv := &mocks.Resources{}
				rsrv.On("Start", mock.Anything, mock.Anything, mock.Anything).Return().Times(1)
				rsrv.On("Summary").Return(sproto.ResourcesSummary{
					AllocationID: a.req.AllocationID,
					ResourcesID:  rID,
					AgentDevices: map[aproto.ID][]device.Device{aproto.ID(agentID): nil},
				})
				rsrv.On("Kill", mock.Anything).Return()
				return rsrv
			}

			resources := []sproto.Resources{
				mockRsvn(sproto.ResourcesID(cproto.NewID()), "agent-1"),
				mockRsvn(sproto.ResourcesID(cproto.NewID()), "agent-2"),
			}
			db.On("AddAllocation", mock.Anything).Return(nil)
			db.On("StartAllocationSession", a.req.AllocationID).Return("", nil)
			trialImpl.Expect(fmt.Sprintf("%T", BuildTaskSpec{}), actors.MockResponse{
				Msg: tasks.TaskSpec{},
			})
			require.NoError(t, system.Ask(rm, actors.ForwardThroughMock{
				To: self,
				Msg: sproto.ResourcesAllocated{
					ID:           a.req.AllocationID,
					ResourcePool: "default",
					Resources:    resources,
				},
			}).Error())
			system.Ask(rm, actors.ForwardThroughMock{To: self, Msg: actor.Ping{}}).Get()
			require.Nil(t, trialImpl.AssertExpectations())
			require.True(t, db.AssertExpectations(t))

			// Pre-ready stage.
			for _, r := range resources {
				summary := r.Summary()
				containerStateChanged := sproto.ResourcesStateChanged{
					ResourcesID:    summary.ResourcesID,
					ResourcesState: sproto.Assigned,
				}
				db.On("UpdateAllocationState", mock.Anything).Return(nil)
				require.NoError(t, system.Ask(self, containerStateChanged).Error())

				beforePulling := time.Now().UTC().Truncate(time.Millisecond)
				containerStateChanged.ResourcesState = sproto.Pulling
				db.On("UpdateAllocationStartTime", mock.Anything).Return(nil)
				require.NoError(t, system.Ask(self, containerStateChanged).Error())
				afterPulling := time.Now().UTC().Truncate(time.Millisecond)
				outOfRange := a.model.StartTime.Before(beforePulling) || a.model.StartTime.After(afterPulling)
				require.Falsef(t, outOfRange,
					"Expected start time of open allocation should be in between %s and %s but it is = %s instead",
					beforePulling, afterPulling, a.model.StartTime)

				containerStateChanged.ResourcesState = sproto.Starting
				require.NoError(t, system.Ask(self, containerStateChanged).Error())
				containerStateChanged.ResourcesState = sproto.Running
				containerStateChanged.ResourcesStarted = &sproto.ResourcesStarted{
					Addresses: []cproto.Address{
						{
							ContainerIP:   "172.0.0.3",
							ContainerPort: 1734,
							HostIP:        "0.0.0.0",
							HostPort:      1734,
						},
					},
				}
				require.NoError(t, system.Ask(self, containerStateChanged).Error())
				containerStateChanged.ResourcesStarted = nil
				require.NoError(t, system.Ask(self, WatchRendezvousInfo{
					ResourcesID: r.Summary().ResourcesID,
				}).Error())
			}
			require.True(t, a.rendezvous.ready())

			// Good stop.
			if tc.acked {
				system.Ask(self, AckPreemption{AllocationID: a.model.AllocationID}).Get()
			}

			// Terminating stage.
			db.On("DeleteAllocationSession", a.model.AllocationID).Return(nil)
			db.On("CompleteAllocation", mock.Anything).Return(nil)
			db.On("CompleteAllocationTelemetry", a.model.AllocationID).Return([]byte{}, nil)
			for _, r := range resources {
				summary := r.Summary()
				containerStateChanged := sproto.ResourcesStateChanged{
					ResourcesID:    summary.ResourcesID,
					ResourcesState: sproto.Terminated,
					ResourcesStopped: &sproto.ResourcesStopped{
						Failure: tc.err,
					},
				}
				require.NoError(t, system.Ask(self, containerStateChanged).Error())
			}
			system.Ask(rm, actors.ForwardThroughMock{To: self, Msg: actor.Ping{}}).Get()
			require.NoError(t, self.AwaitTermination())
			require.True(t, a.exited)
			system.Ask(trial, actor.Ping{}).Get()
			for _, m := range trialImpl.Messages {
				// Just clear the state since it's really hard to check (has random stuff in it).
				if exit, ok := m.(*AllocationExited); ok {
					exit.FinalState = AllocationState{}
				}
			}
			require.Contains(t, trialImpl.Messages, tc.exit)
			require.True(t, db.AssertExpectations(t))
		})
	}
}

func TestAllocationAllGather(t *testing.T) {
	system, _, rm, trialImpl, _, db, a, self := setup(t)

	// Pre-allocated stage.
	mockRsvn := func(cID cproto.ID, agentID string) sproto.Reservation {
		rsrv := &mocks.Reservation{}
		rsrv.On("Start", mock.Anything, mock.Anything, mock.Anything).Return().Times(1)
		rsrv.On("Summary").Return(sproto.ContainerSummary{
			AllocationID: a.req.AllocationID,
			ID:           cID,
			Agent:        agentID,
		})
		rsrv.On("Kill", mock.Anything).Return()
		return rsrv
	}

	reservations := []sproto.Reservation{
		mockRsvn(cproto.NewID(), "agent-1"),
		mockRsvn(cproto.NewID(), "agent-2"),
	}
	db.On("AddAllocation", mock.Anything).Return(nil)
	db.On("StartAllocationSession", a.req.AllocationID).Return("", nil)
	trialImpl.Expect(fmt.Sprintf("%T", BuildTaskSpec{}), actors.MockResponse{
		Msg: tasks.TaskSpec{},
	})
	require.NoError(t, system.Ask(rm, actors.ForwardThroughMock{
		To: self,
		Msg: sproto.ResourcesAllocated{
			ID:           a.req.AllocationID,
			ResourcePool: "default",
			Reservations: reservations,
		},
	}).Error())
	system.Ask(rm, actors.ForwardThroughMock{To: self, Msg: actor.Ping{}}).Get()
	require.Nil(t, trialImpl.AssertExpectations())
	require.True(t, db.AssertExpectations(t))

	numPeers := 4
	type peerState struct {
		uuid    uuid.UUID
		data    *structpb.Struct
		watcher AllGatherWatcher
	}
	peerStates := map[int]*peerState{}

	var expectedData []float64
	for i := 0; i < numPeers; i++ {
		expectedData = append(expectedData, float64(i))
		data, err := structpb.NewStruct(map[string]interface{}{"key": i})
		require.NoError(t, err)

		peerStates[i] = &peerState{
			uuid: uuid.New(),
			data: data,
		}

		resp := system.Ask(self, WatchAllGather{
			WatcherID: peerStates[i].uuid,
			NumPeers:  numPeers,
			Data:      peerStates[i].data,
		})
		require.NoError(t, resp.Error())

		if i == 0 {
			// All gather should now be "started".
			require.NotNil(t, a.allGather)
		}

		require.IsType(t, AllGatherWatcher{}, resp.Get())
		tResp := resp.Get().(AllGatherWatcher)
		peerStates[i].watcher = tResp
	}

	// All gather should now be "completed".
	require.Nil(t, a.allGather)

	for _, ps := range peerStates {
		resp := <-ps.watcher.C
		require.NoError(t, resp.Err)
		require.Len(t, resp.Data, numPeers)

		var actualData []float64
		for _, d := range resp.Data {
			for _, v := range d.AsMap() {
				actualData = append(actualData, v.(float64))
			}
		}

		require.Equal(t, expectedData, actualData)
	}
}

func setup(t *testing.T) (
	*actor.System, *actors.MockActor, *actor.Ref, *actors.MockActor,
	*actor.Ref, *mocks.DB, *Allocation, *actor.Ref,
) {
	require.NoError(t, etc.SetRootPath("../static/srv"))
	system := actor.NewSystem("system")

	// mock resource manager.
	rmImpl := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	rm := system.MustActorOf(actor.Addr("rm"), &rmImpl)

	// mock trial
	loggerImpl := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	loggerAddr := "logger"
	loggerActor := system.MustActorOf(actor.Addr(loggerAddr), &loggerImpl)
	logger := NewCustomLogger(loggerActor)

	// mock trial
	trialImpl := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	trialAddr := "trial"
	trial := system.MustActorOf(actor.Addr(trialAddr), &trialImpl)

	// mock db.
	db := &mocks.DB{}

	// instantiate the allocation
	rID := model.NewRequestID(rand.Reader)
	taskID := model.TaskID(fmt.Sprintf("%s-%s", model.TaskTypeTrial, rID))
	a := NewAllocation(
		sproto.AllocateRequest{
			TaskID:       taskID,
			AllocationID: model.AllocationID(fmt.Sprintf("%s.0", taskID)),
			SlotsNeeded:  2,
			Preemptible:  true,
			DoRendezvous: true,
			// ...
		},
		db,
		rm,
		logger,
	)
	self := system.MustActorOf(actor.Addr(trialAddr, "allocation"), a)
	// Pre-scheduled stage.
	system.Ask(self, actor.Ping{}).Get()
	require.Contains(t, rmImpl.Messages, a.(*Allocation).req)

	return system, &rmImpl, rm, &trialImpl, trial, db, a.(*Allocation), self
}
