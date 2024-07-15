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
