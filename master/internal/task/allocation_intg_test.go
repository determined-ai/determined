//go:build integration
// +build integration

package task

import (
	"fmt"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/internal/portregistry"
	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/task/preemptible"

	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/tasks"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/etc"
	detLogger "github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
)

type mockTaskSpecifier struct{}

func (m mockTaskSpecifier) ToTaskSpec() (t tasks.TaskSpec) { return t }

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
			err:   &sproto.ResourcesFailure{FailureType: sproto.ResourcesFailed},
			exit:  &AllocationExited{Err: sproto.ResourcesFailure{FailureType: sproto.ResourcesFailed}},
		},
		{
			name:  "container failed, but acked preemption",
			acked: true,
			err:   &sproto.ResourcesFailure{FailureType: sproto.ResourcesFailed},
			exit:  &AllocationExited{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			system, _, rm, trialImpl, trial, _, a, self := setup(t)

			// Pre-allocated stage.
			mockRsvn := func(rID sproto.ResourcesID, agentID string) sproto.Resources {
				rsrv := &mocks.Resources{}
				rsrv.On("Start", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).Times(1)
				rsrv.On("Summary").Return(sproto.ResourcesSummary{
					AllocationID:  a.req.AllocationID,
					ResourcesID:   rID,
					ResourcesType: sproto.ResourcesTypeDockerContainer,
					AgentDevices:  map[aproto.ID][]device.Device{aproto.ID(agentID): nil},
				})
				rsrv.On("Kill", mock.Anything, mock.Anything).Return()
				return rsrv
			}

			rID1, rID2 := sproto.ResourcesID(cproto.NewID()), sproto.ResourcesID(cproto.NewID())
			resources := map[sproto.ResourcesID]sproto.Resources{
				rID1: mockRsvn(rID1, "agent-1"),
				rID2: mockRsvn(rID2, "agent-2"),
			}
			require.NoError(t, system.Ask(rm.Ref(), actors.ForwardThroughMock{
				To: self,
				Msg: sproto.ResourcesAllocated{
					ID:           a.req.AllocationID,
					ResourcePool: "default",
					Resources:    resources,
				},
			}).Error())
			system.Ask(rm.Ref(), actors.ForwardThroughMock{To: self, Msg: actor.Ping{}}).Get()
			require.Nil(t, trialImpl.AssertExpectations())

			// Pre-ready stage.
			first := true
			for _, r := range resources {
				summary := r.Summary()
				containerStateChanged := sproto.ResourcesStateChanged{
					ResourcesID:    summary.ResourcesID,
					ResourcesState: sproto.Assigned,
				}
				require.NoError(t, system.Ask(self, containerStateChanged).Error())

				beforePulling := time.Now().UTC().Truncate(time.Millisecond)
				containerStateChanged.ResourcesState = sproto.Pulling
				require.NoError(t, system.Ask(self, containerStateChanged).Error())
				afterPulling := time.Now().UTC().Truncate(time.Millisecond)

				if first {
					outOfRange := a.model.StartTime.Before(beforePulling) ||
						a.model.StartTime.After(afterPulling)
					require.Falsef(t, outOfRange,
						"Expected start time of open allocation should be in between"+
							" %s and %s but it is = %s instead",
						beforePulling, afterPulling, a.model.StartTime)
					first = false
				}

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
				preemptible.Acknowledge(a.model.AllocationID.String())
			}

			// Terminating stage.
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
			system.Ask(rm.Ref(), actors.ForwardThroughMock{To: self, Msg: actor.Ping{}}).Get()
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
			// require.True(t, db.AssertExpectations(t))
		})
	}
}

func setup(t *testing.T) (
	*actor.System, *actors.MockActor, rm.ResourceManager, *actors.MockActor,
	*actor.Ref, *db.PgDB, *Allocation, *actor.Ref,
) {
	require.NoError(t, etc.SetRootPath("../static/srv"))
	system := actor.NewSystem("system")
	portregistry.InitPortRegistry()

	// mock resource manager.
	rmActor := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	rm := actorrm.Wrap(system.MustActorOf(actor.Addr("rm"), &rmActor))

	// mock trial
	trialImpl := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	trialAddr := "trial"
	trial := system.MustActorOf(actor.Addr(trialAddr), &trialImpl)

	// real db.
	pgDB := db.MustSetupTestPostgres(t)

	// instantiate the allocation
	task := db.RequireMockTask(t, pgDB, nil)

	a := NewAllocation(
		detLogger.Context{},
		sproto.AllocateRequest{
			TaskID:       task.TaskID,
			AllocationID: model.AllocationID(fmt.Sprintf("%s.0", task.TaskID)),
			SlotsNeeded:  2,
			Preemptible:  true,
			// ...
		},
		pgDB,
		rm,
		mockTaskSpecifier{},
	)
	self := system.MustActorOf(actor.Addr(trialAddr, "allocation"), a)
	// Pre-scheduled stage.
	system.Ask(self, actor.Ping{}).Get()
	require.Contains(t, rmActor.Messages, a.(*Allocation).req)

	return system, &rmActor, rm, &trialImpl, trial, pgDB, a.(*Allocation), self
}
