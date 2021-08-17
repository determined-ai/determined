package task

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

func TestAllocation(t *testing.T) {
	cases := []struct {
		name  string
		err   *aproto.ContainerFailure
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
			err:   &aproto.ContainerFailure{FailureType: aproto.ContainerFailed},
			exit:  &AllocationExited{Err: &aproto.ContainerFailure{FailureType: aproto.ContainerFailed}},
		},
		{
			name:  "container failed, but acked preemption",
			acked: true,
			err:   &aproto.ContainerFailure{FailureType: aproto.ContainerFailed},
			exit:  &AllocationExited{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			system, _, rm, trialImpl, trial, db, a, self := setup(t)

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
				Msg: &tasks.TaskSpec{},
			})
			require.NoError(t, system.Ask(rm, actors.ForwardThroughMock{
				To: self,
				Msg: sproto.ResourcesAllocated{
					ID:           a.req.AllocationID,
					ResourcePool: "default",
					Reservations: reservations,
				},
			}).Error())
			system.Ask(rm, actors.ForwardThroughMock{To: self, Msg: sproto.ContainerLog{}}).Get() // sync
			require.Nil(t, trialImpl.AssertExpectations())
			require.True(t, db.AssertExpectations(t))

			// Pre-ready stage.
			for _, r := range reservations {
				containerStateChanged := sproto.TaskContainerStateChanged{
					Container: cproto.Container{
						Parent:  actor.Address{},
						ID:      r.Summary().ID,
						State:   cproto.Assigned,
						Devices: []device.Device{},
					},
				}
				require.NoError(t, system.Ask(self, containerStateChanged).Error())
				containerStateChanged.Container.State = cproto.Pulling
				require.NoError(t, system.Ask(self, containerStateChanged).Error())
				containerStateChanged.Container.State = cproto.Starting
				require.NoError(t, system.Ask(self, containerStateChanged).Error())
				containerStateChanged.Container.State = cproto.Running
				containerStateChanged.ContainerStarted = &sproto.TaskContainerStarted{
					Addresses: []cproto.Address{
						{
							ContainerIP:   "0.0.0.0",
							ContainerPort: 1734,
							HostIP:        r.Summary().Agent,
							HostPort:      1734,
						},
					},
				}
				require.NoError(t, system.Ask(self, containerStateChanged).Error())
				containerStateChanged.ContainerStarted = nil
				require.NoError(t, system.Ask(self, WatchRendezvousInfo{
					AllocationID: a.model.AllocationID,
					ContainerID:  r.Summary().ID,
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
			for _, r := range reservations {
				containerStateChanged := sproto.TaskContainerStateChanged{
					Container: cproto.Container{
						Parent:  actor.Address{},
						ID:      r.Summary().ID,
						State:   cproto.Terminated,
						Devices: []device.Device{},
					},
					ContainerStopped: &sproto.TaskContainerStopped{
						ContainerStopped: aproto.ContainerStopped{
							Failure: tc.err,
						},
					},
				}
				require.NoError(t, system.Ask(self, containerStateChanged).Error())
			}
			system.Ask(rm, actors.ForwardThroughMock{To: self, Msg: sproto.ContainerLog{}}).Get() // sync
			require.NoError(t, self.AwaitTermination())
			require.True(t, a.exited)
			system.Ask(trial, sproto.ContainerLog{}).Get() // sync)
			require.Contains(t, trialImpl.Messages, tc.exit)
			require.True(t, db.AssertExpectations(t))
		})
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
			// ...
		},
		db,
		rm,
	)
	self := system.MustActorOf(actor.Addr(trialAddr, "allocation"), a)

	// Pre-scheduled stage.
	system.Ask(self, sproto.ContainerLog{}).Get() // sync
	require.Contains(t, rmImpl.Messages, a.(*Allocation).req)

	return system, &rmImpl, rm, &trialImpl, trial, db, a.(*Allocation), self
}
