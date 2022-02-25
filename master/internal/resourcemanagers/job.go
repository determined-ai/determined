package resourcemanagers

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

func reduceToJobQInfo(reqs AllocReqs) (map[model.JobID]*job.RMJobInfo, map[model.JobID]*actor.Ref) {
	isAdded := make(map[model.JobID]*job.RMJobInfo)
	jobActors := make(map[model.JobID]*actor.Ref)
	jobsAhead := 0
	for _, req := range reqs {
		if !req.IsUserVisible {
			continue
		}
		v1JobInfo, exists := isAdded[req.JobID]
		if !exists {
			v1JobInfo = &job.RMJobInfo{
				JobsAhead: jobsAhead,
				State:     req.State,
			}
			isAdded[req.JobID] = v1JobInfo
			jobActors[req.JobID] = req.Group
			jobsAhead++
		}
		// Carry over the the highest state.
		if v1JobInfo.State < req.State {
			isAdded[req.JobID].State = req.State
		}
		v1JobInfo.RequestedSlots += req.SlotsNeeded
		if job.ScheduledStates[req.State] {
			v1JobInfo.AllocatedSlots += req.SlotsNeeded
		}
	}
	return isAdded, jobActors
}

func jobStats(taskList *taskList) *jobv1.QueueStats {
	stats := &jobv1.QueueStats{}
	reqs := make(AllocReqs, 0)
	for it := taskList.iterator(); it.next(); {
		req := it.value()
		if !req.IsUserVisible {
			continue
		}
		reqs = append(reqs, req)
	}
	jobsMap, _ := reduceToJobQInfo(reqs)
	for _, jobInfo := range jobsMap {
		if jobInfo.State == job.SchedulingStateQueued {
			stats.QueuedCount++
		} else {
			stats.ScheduledCount++
		}
	}
	return stats
}

// assignmentIsScheduled determines if a resource allocation assignment is considered equivalent to
// being scheduled.
func assignmentIsScheduled(allocatedResources *sproto.ResourcesAllocated) bool {
	return allocatedResources != nil && len(allocatedResources.Reservations) > 0
}

// could be swapped for another job order representation
// 0 or nonexisting keys mean that it needs to initialize.
type jobSortState = map[model.JobID]decimal.Decimal

func initalizeJobSortState() jobSortState {
	state := make(jobSortState)
	state[job.HeadAnchor] = decimal.NewFromInt(0)
	state[job.TailAnchor] = initalizeQueuePosition(time.Now())
	return state
}

func computeNewJobPos(msg job.MoveJob, qPositions jobSortState) (decimal.Decimal, error) {
	if msg.Anchor1 == msg.ID || msg.Anchor2 == msg.ID {
		return decimal.NewFromInt(0), fmt.Errorf("cannot move job relative to itself")
	}

	qPos1, ok := qPositions[msg.Anchor1]
	if !ok {
		return decimal.NewFromInt(0), fmt.Errorf("could not find anchor job %s", msg.Anchor1)
	}

	qPos2, ok := qPositions[msg.Anchor2]
	if !ok {
		return decimal.NewFromInt(0), fmt.Errorf("could not find anchor job %s", msg.Anchor2)
	}

	qPos3, ok := qPositions[msg.ID]
	if !ok {
		return decimal.NewFromInt(0), fmt.Errorf("could not find job %s", msg.ID)
	}

	// check if qPos3 is between qPos1 and qPos2
	smallPos := decimal.Min(qPos1, qPos2)
	//smallPos := math.Min(qPos1, qPos2)
	bigPos := decimal.Max(qPos1, qPos2)
	if qPos3.GreaterThan(smallPos) && qPos3.LessThan(bigPos) {
		return decimal.NewFromInt(0), nil // no op. Job is already in the correct position.
	}

	newPos := decimal.Avg(qPos1, qPos2)

	if newPos.Equal(qPos1) || newPos.Equal(qPos2) {
		return decimal.NewFromInt(0), fmt.Errorf("unable to compute a new job position for job %s", msg.ID)
	}

	return newPos, nil
}

func initalizeQueuePosition(aTime time.Time) decimal.Decimal {
	// we could shift this back and forth to give us more more.
	return decimal.New(aTime.UnixMicro(), 1000)
}

// we might RMs to have easier/faster access to this information than this.
func getJobSubmissionTime(taskList *taskList, jobID model.JobID) (time.Time, error) {
	for it := taskList.iterator(); it.next(); {
		req := it.value()
		if req.JobID == jobID {
			return req.JobSubmissionTime, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not find an active job with id %s", jobID)
}
