package resourcemanagers

import (
	"fmt"
	"math"
	"sort"
	"time"

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

// TODO add a comment.
// adjust in place.
func adjustPriorities(positions jobSortState) jobSortState {
	reverse_positions := map[float64]model.JobID{}
	var positions_list []float64
	for id, position := range positions {
		reverse_positions[position] = id
		positions_list = append(positions_list, position)
	}

	sort.Float64s(positions_list)

	idx := float64(1) //TODO: refactor to use the timestamp of the tail node

	for _, position := range positions_list {
		positions[reverse_positions[position]] = idx
		idx += 1
	}
	return positions
}

// could be swapped for another job order representation
// 0 or nonexisting keys mean that it needs to initialize.
type jobSortState = map[model.JobID]float64

func initalizeJobSortState() jobSortState {
	state := make(jobSortState)
	state[job.HeadAnchor] = 0
	state[job.TailAnchor] = initalizeQueuePosition(time.Now())
	return state
}

func computeNewJobPos(msg job.MoveJob, qPositions jobSortState) (float64, bool, error) {
	if msg.Anchor1 == msg.ID || msg.Anchor2 == msg.ID {
		return 0, false, fmt.Errorf("cannot move job relative to itself")
	}

	qPos1, ok := qPositions[msg.Anchor1]
	if !ok {
		return 0, false, fmt.Errorf("could not find anchor job %s", msg.Anchor1)
	}

	qPos2, ok := qPositions[msg.Anchor2]
	if !ok {
		return 0, false, fmt.Errorf("could not find anchor job %s", msg.Anchor2)
	}

	qPos3, ok := qPositions[msg.ID]
	if !ok {
		return 0, false, fmt.Errorf("could not find job %s", msg.ID)
	}

	// check if qPos3 is between qPos1 and qPos2
	smallPos := math.Min(qPos1, qPos2)
	bigPos := math.Max(qPos1, qPos2)
	if qPos3 > smallPos && qPos3 < bigPos {
		return 0, false, nil // no op. Job is already in the correct position.
	}

	newPos := (qPos1 + qPos2) / 2

	return newPos, false, nil
}

func initalizeQueuePosition(aTime time.Time) float64 {
	// we could shift this back and forth to give us more more.
	return float64(aTime.UnixMicro())
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
