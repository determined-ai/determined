package agentrm

import (
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type (
	SendRequestResourcesToResourceManager  struct{}
	SendResourcesReleasedToResourceManager struct{}
	ThrowError                             struct{}
	ThrowPanic                             struct{}
)

var ErrMock = errors.New("mock error")

type MockTask struct {
	RMRef *actor.Ref

	ID             model.AllocationID
	JobID          string
	Group          *MockGroup
	SlotsNeeded    int
	NonPreemptible bool
	ResourcePool   string
	AllocatedAgent *MockAgent
	// Any test that set this to false is half wrong. It is used as a proxy to oversubscribe agents.
	ContainerStarted  bool
	JobSubmissionTime time.Time
}

func (t *MockTask) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		allocationmap.RegisterAllocation(t.ID, ctx.Self())
	case actor.PostStop:
		allocationmap.UnregisterAllocation(t.ID)
	case SendRequestResourcesToResourceManager:
		task := sproto.AllocateRequest{
			AllocationID:      t.ID,
			JobID:             model.JobID(t.JobID),
			JobSubmissionTime: t.JobSubmissionTime,
			Name:              string(t.ID),
			SlotsNeeded:       t.SlotsNeeded,
			Preemptible:       !t.NonPreemptible,
			ResourcePool:      t.ResourcePool,
			AllocationRef:     ctx.Self(),
		}
		if t.Group == nil {
			task.Group = ctx.Self()
		} else {
			task.Group = ctx.Self().System().Get(actor.Addr(t.Group.ID))
		}
		if ctx.ExpectingResponse() {
			ctx.Respond(ctx.Ask(t.RMRef, task).Get())
		} else {
			ctx.Tell(t.RMRef, task)
		}
	case SendResourcesReleasedToResourceManager:
		task := sproto.ResourcesReleased{AllocationID: t.ID}
		if ctx.ExpectingResponse() {
			ctx.Respond(ctx.Ask(t.RMRef, task).Get())
		} else {
			ctx.Tell(t.RMRef, task)
		}
	case ThrowError:
		return ErrMock
	case ThrowPanic:
		panic(ErrMock)

	case sproto.ResourcesAllocated:
		rank := 0
		for _, allocation := range msg.Resources {
			if err := allocation.Start(ctx, nil, tasks.TaskSpec{}, sproto.ResourcesRuntimeInfo{
				Token:        "",
				AgentRank:    rank,
				IsMultiAgent: len(msg.Resources) > 1,
			}); err != nil {
				ctx.Respond(err)
				return nil
			}
			rank++
		}
	case sproto.ReleaseResources:
		ctx.Tell(t.RMRef, sproto.ResourcesReleased{AllocationID: t.ID})

	case sproto.ResourcesStateChanged:

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

type MockGroup struct {
	ID       string
	MaxSlots *int
	Weight   float64
	Priority *int
}

func (g *MockGroup) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart:
	case actor.PostStop:
	case *sproto.RMJobInfo:
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func MockTaskToAllocateRequest(
	mockTask *MockTask, allocationRef *actor.Ref,
) *sproto.AllocateRequest {
	jobID := mockTask.JobID
	jobSubmissionTime := mockTask.JobSubmissionTime

	if jobID == "" {
		jobID = string(mockTask.ID)
	}
	if jobSubmissionTime.IsZero() {
		jobSubmissionTime = allocationRef.RegisteredTime()
	}

	req := &sproto.AllocateRequest{
		AllocationID:      mockTask.ID,
		JobID:             model.JobID(jobID),
		SlotsNeeded:       mockTask.SlotsNeeded,
		IsUserVisible:     true,
		AllocationRef:     allocationRef,
		Preemptible:       !mockTask.NonPreemptible,
		JobSubmissionTime: jobSubmissionTime,
	}
	return req
}

type MockAgent struct {
	ID                    string
	Slots                 int
	SlotsUsed             int
	MaxZeroSlotContainers int
	ZeroSlotContainers    int
}

func NewMockAgent(
	id string,
	slots int,
	slotsUsed int,
	maxZeroSlotContainers int,
	zeroSlotContainers int,
) *MockAgent {
	return &MockAgent{
		ID:                    id,
		Slots:                 slots,
		SlotsUsed:             slotsUsed,
		MaxZeroSlotContainers: maxZeroSlotContainers,
		ZeroSlotContainers:    zeroSlotContainers,
	}
}

func (m *MockAgent) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart:
	case actor.PostStop:
	case sproto.StartTaskContainer:
	case sproto.KillTaskContainer:
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
