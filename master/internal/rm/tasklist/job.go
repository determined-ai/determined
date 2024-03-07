package tasklist

import (
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

var invalidJobQPos = decimal.NewFromInt(0)

// ReduceToJobQInfo takes a list of AllocateRequest and reduces it to a summary of the Job Queue.
func ReduceToJobQInfo(reqs AllocReqs) map[model.JobID]*sproto.RMJobInfo {
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
		// Carry over the highest state.
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

// JobStats returns quick job-related stats about the TaskList.
func JobStats(taskList *TaskList) *jobv1.QueueStats {
	reqs := make(AllocReqs, 0)
	for it := taskList.Iterator(); it.Next(); {
		req := it.Value()
		if !req.IsUserVisible {
			continue
		}
		reqs = append(reqs, req)
	}
	stats := requestsToQueueStats(reqs)
	return stats
}

// JobStatsByPool returns quick job-related stats about the TaskList, by resource pool.
func JobStatsByPool(taskList *TaskList, resourcePool string) *jobv1.QueueStats {
	reqs := make(AllocReqs, 0)
	for it := taskList.Iterator(); it.Next(); {
		req := it.Value()
		if !req.IsUserVisible || req.ResourcePool != resourcePool {
			continue
		}
		reqs = append(reqs, req)
	}
	return requestsToQueueStats(reqs)
}

func requestsToQueueStats(reqs []*sproto.AllocateRequest) *jobv1.QueueStats {
	jobsMap := ReduceToJobQInfo(reqs)
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

// AssignmentIsScheduled determines if a resource allocation assignment is considered equivalent to
// being scheduled.
func AssignmentIsScheduled(allocatedResources *sproto.ResourcesAllocated) bool {
	return allocatedResources != nil
}

// JobSortState models a job queue, and the positions of all jobs within it.
type JobSortState map[model.JobID]decimal.Decimal

// SetJobPosition sets the job position in the queue, relative to the anchors.
func (j JobSortState) SetJobPosition(
	jobID model.JobID,
	anchor1 model.JobID,
	anchor2 model.JobID,
	aheadOf bool,
	isK8s bool,
) (decimal.Decimal, error) {
	newPos, err := computeNewJobPos(jobID, anchor1, anchor2, j)
	if err != nil {
		return decimal.Decimal{}, err
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

	j[sproto.TailAnchor] = InitializeQueuePosition(time.Now(), isK8s)
	j[jobID] = newPos

	return newPos, nil
}

// RecoverJobPosition explicitly sets the position of a job.
func (j JobSortState) RecoverJobPosition(jobID model.JobID, position decimal.Decimal) {
	j[jobID] = position
}

// InitializeJobSortState constructs a JobSortState based on the RM type.
func InitializeJobSortState(isK8s bool) JobSortState {
	state := make(JobSortState)
	if isK8s {
		state[sproto.HeadAnchor] = decimal.New(1, sproto.K8sExp)
	} else {
		state[sproto.HeadAnchor] = decimal.New(1, sproto.DecimalExp)
	}
	state[sproto.TailAnchor] = InitializeQueuePosition(time.Now(), isK8s)
	return state
}

func computeNewJobPos(
	jobID model.JobID,
	anchor1 model.JobID,
	anchor2 model.JobID,
	qPositions JobSortState,
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

// InitializeQueuePosition constructs a new queue position from time and RM type.
func InitializeQueuePosition(aTime time.Time, isK8s bool) decimal.Decimal {
	// we could add exponent to give us more insertions if needed.
	if isK8s {
		return decimal.New(aTime.UnixMicro(), sproto.K8sExp)
	}
	return decimal.New(aTime.UnixMicro(), sproto.DecimalExp)
}

// GetJobSubmissionTime returns the submission time for the first task found in the list for a job.
// we might RMs to have easier/faster access to this information than this.
func GetJobSubmissionTime(taskList *TaskList, jobID model.JobID) (time.Time, error) {
	for it := taskList.Iterator(); it.Next(); {
		req := it.Value()
		if req.JobID == jobID {
			return req.JobSubmissionTime, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not find an active job with id %s", jobID)
}

// FindAnchor finds a second anchor and its priority and determines if the moving job needs a
// priority change to move ahead or behind the anchor.
func FindAnchor(
	jobID model.JobID,
	anchorID model.JobID,
	aheadOf bool,
	taskList *TaskList,
	groups map[model.JobID]*Group,
	queuePositions JobSortState,
	k8s bool,
) (bool, model.JobID, int) {
	var secondAnchor model.JobID
	targetPriority := 0
	anchorPriority := 0
	anchorIdx := 0
	prioChange := false

	sortedReqs := SortTasksWithPosition(taskList, groups, queuePositions, k8s)

	for i, req := range sortedReqs {
		if req.JobID == jobID {
			targetPriority = *groups[req.JobID].Priority
		} else if req.JobID == anchorID {
			anchorPriority = *groups[req.JobID].Priority
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

// NeedMove returns true if the jobPos indicates a job needs a move to be ahead of or behind
// the anchorPos.
func NeedMove(
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

// SortTasksWithPosition returns a sorted view of the sproto.AllocateRequest's that make up
// the TaskList, sorted in priority order.
func SortTasksWithPosition(
	taskList *TaskList,
	groups map[model.JobID]*Group,
	jobPositions JobSortState,
	k8s bool,
) []*sproto.AllocateRequest {
	var reqs []*sproto.AllocateRequest
	for it := taskList.Iterator(); it.Next(); {
		req := it.Value()
		group, ok := groups[req.JobID]
		if !ok {
			log.Errorf(
				`found an allocation (%s) without a group (%s) when trying to sort by priority; ignoring it`,
				req.Name, req.JobID,
			)
			continue
		}
		if group.Priority == nil {
			log.Errorf(
				`found an allocation (%s) without a priority (%s) when trying to sort by priority; ignoring it`,
				req.Name, req.JobID,
			)
			continue
		}
		reqs = append(reqs, req)
	}
	sort.Slice(reqs, func(i, j int) bool {
		p1 := *groups[reqs[i].JobID].Priority
		p2 := *groups[reqs[j].JobID].Priority
		if k8s { // in k8s, higher priority == more prioritized
			switch {
			case p1 > p2:
				return true
			case p2 > p1:
				return false
			}
		} else {
			switch {
			case p1 > p2:
				return false
			case p2 > p1:
				return true
			}
		}

		return comparePositions(reqs[i], reqs[j], jobPositions) > 0
	})

	return reqs
}

// comparePositions returns the following:
// 1 if a is in front of b.
// 0 if a is equal to b in position.
// -1 if a is behind b.
func comparePositions(a, b *sproto.AllocateRequest, jobPositions JobSortState) int {
	aPosition, aOk := jobPositions[a.JobID]
	bPosition, bOk := jobPositions[b.JobID]
	zero := decimal.NewFromInt(0)
	if !aOk || !bOk {
		// we shouldn't run into this situation once k8 support is implemented other than
		// when testing.
		return allocationRequestComparator(a, b) * -1
	}
	switch {
	case aPosition == bPosition:
		return allocationRequestComparator(a, b) * -1
	case aPosition.LessThan(zero) || bPosition.LessThan(zero):
		if aPosition.GreaterThan(zero) {
			return 1
		}
		return -1
	case aPosition.LessThan(bPosition):
		return 1
	default:
		return -1
	}
}

// AllocReqs is an alias for a list of Allocate Requests.
type AllocReqs = []*sproto.AllocateRequest
