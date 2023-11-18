package agentrm

import (
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

type (
	SendRequestResourcesToResourceManager  struct{}
	SendResourcesReleasedToResourceManager struct{}
	ThrowError                             struct{}
	ThrowPanic                             struct{}
)

var ErrMock = errors.New("mock error")

type MockTask struct {
	RPRef *resourcePool

	TaskID         model.TaskID
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

	BlockedNodes []string
}

type MockGroup struct {
	ID       string
	MaxSlots *int
	Weight   float64
	Priority *int
}

func MockTaskToAllocateRequest(mockTask *MockTask) *sproto.AllocateRequest {
	jobID := mockTask.JobID
	jobSubmissionTime := mockTask.JobSubmissionTime

	if jobID == "" {
		jobID = string(mockTask.ID)
	}
	if jobSubmissionTime.IsZero() {
		jobSubmissionTime = time.Now()
	}

	req := &sproto.AllocateRequest{
		TaskID:            mockTask.TaskID,
		AllocationID:      mockTask.ID,
		JobID:             model.JobID(jobID),
		SlotsNeeded:       mockTask.SlotsNeeded,
		IsUserVisible:     true,
		Preemptible:       !mockTask.NonPreemptible,
		JobSubmissionTime: jobSubmissionTime,
		BlockedNodes:      mockTask.BlockedNodes,
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
