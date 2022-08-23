package rm

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

var invalidJobQPos = decimal.NewFromInt(0)

func reduceToJobQInfo(reqs AllocReqs) map[model.JobID]*sproto.RMJobInfo {
	isAdded := make(map[model.JobID]*sproto.RMJobInfo)
	jobsAhead := 0
	for _, req := range reqs {
		if !req.IsUserVisible {
			continue
		}
		v1JobInfo, exists := isAdded[req.JobID]
		if !exists {
			v1JobInfo = &sproto.RMJobInfo{
				JobsAhead: jobsAhead,
				State:     req.State,
			}
			isAdded[req.JobID] = v1JobInfo
			jobsAhead++
		}
		// Carry over the the highest state.
		if v1JobInfo.State < req.State {
			isAdded[req.JobID].State = req.State
		}
		v1JobInfo.RequestedSlots += req.SlotsNeeded
		if sproto.ScheduledStates[req.State] {
			v1JobInfo.AllocatedSlots += req.SlotsNeeded
		}
	}
	return isAdded
}

func jobStats(taskList *taskList) *jobv1.QueueStats {
	reqs := make(AllocReqs, 0)
	for it := taskList.iterator(); it.next(); {
		req := it.value()
		if !req.IsUserVisible {
			continue
		}
		reqs = append(reqs, req)
	}
	stats := requestsToQueueStats(reqs)
	return stats
}

//nolint:deadcode,nolintlint // Method used by Slurm support in determined-ee
func jobStatsByPool(taskList *taskList, resourcePool string) *jobv1.QueueStats {
	reqs := make(AllocReqs, 0)
	for it := taskList.iterator(); it.next(); {
		req := it.value()
		if !req.IsUserVisible || req.ResourcePool != resourcePool {
			continue
		}
		reqs = append(reqs, req)
	}
	return requestsToQueueStats(reqs)
}

func requestsToQueueStats(reqs []*sproto.AllocateRequest) *jobv1.QueueStats {
	jobsMap := reduceToJobQInfo(reqs)
	stats := &jobv1.QueueStats{}
	for _, jobInfo := range jobsMap {
		if jobInfo.State == sproto.SchedulingStateQueued {
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
	return allocatedResources != nil
}

type jobSortState map[model.JobID]decimal.Decimal

func (j jobSortState) SetJobPosition(
	jobID model.JobID,
	anchor1 model.JobID,
	anchor2 model.JobID,
	aheadOf bool,
	isK8s bool,
) (sproto.RegisterJobPosition, error) {
	newPos, err := computeNewJobPos(jobID, anchor1, anchor2, j)
	if err != nil {
		return sproto.RegisterJobPosition{}, err
	}

	// if the calculated position results in the wrong order
	// we subtract a minimal decimal amount instead.
	minDecimal := decimal.New(1, sproto.DecimalExp)
	if isK8s {
		minDecimal = decimal.New(1, sproto.K8sExp)
	}
	if aheadOf && newPos.GreaterThanOrEqual(j[anchor1]) {
		newPos = j[anchor1].Sub(minDecimal)
	} else if !aheadOf && newPos.LessThanOrEqual(j[anchor1]) {
		newPos = j[anchor1].Add(minDecimal)
	}

	j[sproto.TailAnchor] = initalizeQueuePosition(time.Now(), isK8s)
	j[jobID] = newPos

	return sproto.RegisterJobPosition{
		JobID:       jobID,
		JobPosition: newPos,
	}, nil
}

func (j jobSortState) RecoverJobPosition(jobID model.JobID, position decimal.Decimal) {
	j[jobID] = position
}

func initalizeJobSortState(isK8s bool) jobSortState {
	state := make(jobSortState)
	if isK8s {
		state[sproto.HeadAnchor] = decimal.New(1, sproto.K8sExp)
	} else {
		state[sproto.HeadAnchor] = decimal.New(1, sproto.DecimalExp)
	}
	state[sproto.TailAnchor] = initalizeQueuePosition(time.Now(), isK8s)
	return state
}

func computeNewJobPos(
	jobID model.JobID,
	anchor1 model.JobID,
	anchor2 model.JobID,
	qPositions jobSortState,
) (decimal.Decimal, error) {
	if anchor1 == jobID || anchor2 == jobID {
		return invalidJobQPos, fmt.Errorf("cannot move job relative to itself")
	}

	qPos1, ok := qPositions[anchor1]
	if !ok {
		return invalidJobQPos, fmt.Errorf("could not find anchor job %s", anchor1)
	}

	qPos2, ok := qPositions[anchor2]
	if !ok {
		return invalidJobQPos, fmt.Errorf("could not find anchor job %s", anchor2)
	}

	qPos3, ok := qPositions[jobID]
	if !ok {
		return invalidJobQPos, fmt.Errorf("could not find job %s", jobID)
	}

	// check if qPos3 is between qPos1 and qPos2
	smallPos := decimal.Min(qPos1, qPos2)
	bigPos := decimal.Max(qPos1, qPos2)
	if qPos3.GreaterThan(smallPos) && qPos3.LessThan(bigPos) {
		return qPos3, nil // no op. Job is already in the correct position.
	}

	newPos := decimal.Avg(qPos1, qPos2)

	if newPos.Equal(qPos1) || newPos.Equal(qPos2) {
		return invalidJobQPos, fmt.Errorf("unable to compute a new job position for job %s",
			jobID)
	}

	return newPos, nil
}

func initalizeQueuePosition(aTime time.Time, isK8s bool) decimal.Decimal {
	// we could add exponent to give us more insertions if needed.
	if isK8s {
		return decimal.New(aTime.UnixMicro(), sproto.K8sExp)
	}
	return decimal.New(aTime.UnixMicro(), sproto.DecimalExp)
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

func findAnchor(
	jobID model.JobID,
	anchorID model.JobID,
	aheadOf bool,
	taskList *taskList,
	groups map[*actor.Ref]*group,
	queuePositions jobSortState,
	k8s bool,
) (bool, model.JobID, int) {
	var secondAnchor model.JobID
	targetPriority := 0
	anchorPriority := 0
	anchorIdx := 0
	prioChange := false

	sortedReqs := sortTasksWithPosition(taskList, groups, queuePositions, k8s)

	for i, req := range sortedReqs {
		if req.JobID == jobID {
			targetPriority = *groups[req.Group].priority
		} else if req.JobID == anchorID {
			anchorPriority = *groups[req.Group].priority
			anchorIdx = i
		}
	}

	if aheadOf {
		if anchorIdx == 0 {
			secondAnchor = sproto.HeadAnchor
		} else {
			secondAnchor = sortedReqs[anchorIdx-1].JobID
		}
	} else {
		if anchorIdx >= len(sortedReqs)-1 {
			secondAnchor = sproto.TailAnchor
		} else {
			secondAnchor = sortedReqs[anchorIdx+1].JobID
		}
	}

	if targetPriority != anchorPriority {
		prioChange = true
	}

	return prioChange, secondAnchor, anchorPriority
}

func needMove(
	jobPos decimal.Decimal,
	anchorPos decimal.Decimal,
	secondPos decimal.Decimal,
	aheadOf bool,
) bool {
	if aheadOf {
		if jobPos.LessThan(anchorPos) && jobPos.GreaterThan(secondPos) {
			return false
		}
		return true
	}
	if jobPos.GreaterThan(anchorPos) && jobPos.LessThan(secondPos) {
		return false
	}

	return true
}
