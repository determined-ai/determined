package resourcemanagers

import (
	"fmt"
	"math"
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
func adjustPriorities(positions *jobSortState) {
	// max available space: 0 to current time.
	// prev := float64(0)
	// i := 0

	// TODO make sure the highest qposition doesn't increase.. even when moving jobs

	// for i < len(reqs) {
	// 	if groups[reqs[i].Group].qPosition == -1 {
	// 		if i == len(reqs)-1 {
	// 			groups[reqs[i].Group].qPosition = prev + 1
	// 		} else {
	// 			// seek the next populated value
	// 			var seeker int
	// 			for loc := i + 1; loc < len(reqs); loc++ {
	// 				seeker = loc
	// 				if groups[reqs[loc].Group].qPosition != -1 {
	// 					break
	// 				} else if loc == len(reqs) {
	// 					break
	// 				}
	// 			}

	// 			// set queue positions
	// 			if seeker >= len(reqs)-1 {
	// 				for setter := i; setter < len(reqs); setter++ {
	// 					groups[reqs[setter].Group].qPosition = prev + 1
	// 					prev++
	// 				}
	// 			} else {
	// 				maxValue := groups[reqs[seeker].Group].qPosition
	// 				diff := float64(seeker - i)
	// 				increment := (maxValue - prev) / diff

	// 				for setter := i; setter < seeker; setter++ {
	// 					groups[reqs[setter].Group].qPosition = prev + increment
	// 					prev += increment
	// 				}
	// 			} // find the value of the non negative position and add increments
	// 		}
	// 	}
	// 	prev = groups[reqs[i].Group].qPosition
	// 	i++
	// }
}

// could be swapped for another job order representation
// 0 or nonexisting keys mean that it needs to initialize.
type jobSortState = map[model.JobID]float64

func initalizeJobSortState() jobSortState {
	state := make(jobSortState)
	state[job.HeadAnchor] = 0
	// FIXME this can't go over the current time. job.TailAnchor
	state[job.TailAnchor] = math.MaxFloat64
	return state
}

func computeNewJobPos(msg job.MoveJob, qPositions jobSortState) (float64, bool, error) {
	if msg.Anchor1 == msg.ID || msg.Anchor2 == msg.ID {
		return 0, false, fmt.Errorf("cannot move job relative to itself")
	}
	// find what that position of the anchor job is
	// anchorInfo, ok := jobQ[msg.Anchor]
	// if !ok {
	// 	return 0, fmt.Errorf("could not find anchor job %s", msg.Anchor)
	// }

	// get q positions for anchor, anchor.next or before
	qPos1, ok := qPositions[msg.Anchor1]
	if !ok {
		return 0, false, fmt.Errorf("could not find anchor job %s", msg.Anchor1)
	}

	// FIXME this can't go over the current time. job.TailAnchor
	qPos2, ok := qPositions[msg.Anchor2]
	if !ok {
		return 0, false, fmt.Errorf("could not find anchor job %s", msg.Anchor2)
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
