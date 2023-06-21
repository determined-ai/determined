package sproto

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// DecimalExp is a constant used by decimal.Decimal objects to denote its exponent.
const DecimalExp = 1000

// K8sExp is a constant used by decimal.Decimal objects to denote the exponent for Kubernetes
// labels as k8s labels are limited to 63 characters.
const K8sExp = 30

var (
	// HeadAnchor is an internal anchor for the head of the job queue.
	HeadAnchor = model.JobID("INTERNAL-head")
	// TailAnchor is an internal anchor for the tail of the job queue.
	TailAnchor = model.JobID("INTERNAL-tail")
)

// AQueue is a map of jobID to RMJobInfo.
type AQueue = map[model.JobID]*RMJobInfo

// RMJobInfo packs information available only to the RM that updates frequently.
type RMJobInfo struct { // rename ?
	JobsAhead      int
	State          SchedulingState
	RequestedSlots int
	AllocatedSlots int
}

// GetJobSummary requests a summary of the job.
type GetJobSummary struct {
	JobID        model.JobID
	ResourcePool string
}

// GetJob requests a job representation from a job.
type GetJob struct{}

// GetJobQ is used to get all job information in one go to avoid any inconsistencies.
type GetJobQ struct {
	ResourcePool string
}

// DeleteJob instructs the RM to clean up all metadata associated with a job external to
// Determined.
type DeleteJob struct {
	JobID model.JobID
}

// DeleteJobResponse returns to the caller if the cleanup was successful or not.
type DeleteJobResponse struct {
	Err <-chan error
}

// EmptyDeleteJobResponse returns a response with an empty error chan.
func EmptyDeleteJobResponse() DeleteJobResponse {
	return DeleteJobResponseOf(nil)
}

// DeleteJobResponseOf returns a response containing the specified error.
func DeleteJobResponseOf(input error) DeleteJobResponse {
	respC := make(chan error, 1)
	respC <- input
	return DeleteJobResponse{Err: respC}
}

// GetJobQStats requests stats for a queue.
// Expected response: jobv1.QueueStats.
type GetJobQStats struct {
	ResourcePool string
}

type (
	// SetGroupWeight sets the weight of a group in the fair share scheduler.
	SetGroupWeight struct {
		Weight       float64
		ResourcePool string
		Handler      *actor.Ref
	}
	// SetGroupPriority sets the priority of the group in the priority scheduler.
	SetGroupPriority struct {
		Priority     int
		ResourcePool string
		Handler      *actor.Ref
	}
	// SetResourcePool switches the resource pool that the job belongs to.
	SetResourcePool struct {
		ResourcePool string
		Handler      *actor.Ref
	}
	// MoveJob requests the job to be moved within a priority queue relative to another job.
	MoveJob struct {
		ID     model.JobID
		Anchor model.JobID
		Ahead  bool
	}
)

// RegisterJobPosition gets sent from the resource pool to experiment/command actors.
// It notifies the task of its new position.
type RegisterJobPosition struct {
	JobID       model.JobID
	JobPosition decimal.Decimal
}

// RecoverJobPosition gets sent from the experiment or command actor to the resource pool.
// Notifies the resource pool of the position of the job.
type RecoverJobPosition struct {
	JobID        model.JobID
	JobPosition  decimal.Decimal
	ResourcePool string
}

// SchedulingState denotes the scheduling state of a job and in order of its progression value.
type SchedulingState uint8 // CHECK perhaps could be defined in resource manager. cyclic import

const (
	// SchedulingStateQueued denotes a queued job waiting to be scheduled.
	SchedulingStateQueued SchedulingState = 0
	// SchedulingStateScheduledBackfilled denotes a job that is scheduled for execution as a backfill.
	SchedulingStateScheduledBackfilled SchedulingState = 1
	// SchedulingStateScheduled denotes a job that is scheduled for execution.
	SchedulingStateScheduled SchedulingState = 2
)

// Proto returns proto representation of SchedulingState.
func (s SchedulingState) Proto() jobv1.State {
	switch s {
	case SchedulingStateQueued:
		return jobv1.State_STATE_QUEUED
	case SchedulingStateScheduledBackfilled:
		return jobv1.State_STATE_SCHEDULED_BACKFILLED
	case SchedulingStateScheduled:
		return jobv1.State_STATE_SCHEDULED
	default:
		return jobv1.State_STATE_UNSPECIFIED
	}
}

// SchedulingStateFromProto returns SchedulingState from proto representation.
func SchedulingStateFromProto(state jobv1.State) SchedulingState {
	switch state {
	case jobv1.State_STATE_QUEUED:
		return SchedulingStateQueued
	case jobv1.State_STATE_SCHEDULED_BACKFILLED:
		return SchedulingStateScheduledBackfilled
	case jobv1.State_STATE_SCHEDULED:
		return SchedulingStateScheduled
	default:
		panic("unexpected state")
	}
}

// ScheduledStates provides a list of ScheduledStates that are considered scheduled.
var ScheduledStates = map[SchedulingState]bool{
	SchedulingStateScheduled:           true,
	SchedulingStateScheduledBackfilled: true,
}

// ErrJobNotFound returns a standard job error.
func ErrJobNotFound(jobID model.JobID) error {
	return fmt.Errorf("job %s not found", jobID)
}
