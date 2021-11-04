package resourcemanagers

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// FIXME haven't decided if resource manager actor should be responsible for this or not
// we don't want a separate actor do we? could be useful for streaming job endpoints.
// CHECK do we define the following messages in sproto package?
// QUESTION should we use proto defined messages more often internally or keep them at api level

// SetJobOrder conveys a job queue change for a specific jobID to the resource pool.
type SetJobOrder struct {
	QPosition float64
	Weight    float64
	Priority  *int
	JobID     model.JobID
}

// GetJobOrder requests a list of *jobv1.Job.
// Expected response: []*jobv1.Job.
// DEPRECATED
type GetJobOrder struct{}

// GetJobQInfo is used to get all job information in one go to avoid any inconsistencies.
type GetJobQInfo struct{}

// GetJobQStats requests stats for a queue.
// Expected response: jobv1.QueueStats.
type GetJobQStats struct{}

// allocateReqToV1Job partially fills a jobv1.Job where the information is locally available.
/*
information we need out of scheduler, RP, and RM:
- order of jobs, jobsAhead. (just need to find out what job allocreq belongs to)
- state of jobs. sheduling state
- slots: acq, requested

task actors get ResourcesAllocated & released messages and we can compute state of jobs
and busy slots from there.
*/
func allocateReqToV1Job( // TODO deprecate (k8)
	group *group,
	schedulerType string,
	req *sproto.AllocateRequest,
	jobsAhead int,
) (job *jobv1.Job) {
	if req.Job == nil {
		return nil
	}
	var submissionTime *timestamp.Timestamp
	if req.Job.JobType != model.JobTypeExperiment && req.TaskActor != nil {
		submissionTime = timestamppb.New(req.TaskActor.RegisteredTime())
	}

	job = &jobv1.Job{
		JobId: string(req.Job.JobID),
		Summary: &jobv1.JobSummary{
			State:     req.Job.State.Proto(),
			JobsAhead: int32(jobsAhead),
		},
		EntityId:       req.Job.EntityID,
		Type:           req.Job.JobType.Proto(),
		IsPreemptible:  req.Preemptible,
		ResourcePool:   req.ResourcePool,
		SubmissionTime: submissionTime,
		RequestedSlots: int32(req.Job.RequestedSlots),
		AllocatedSlots: int32(req.Job.AllocatedSlots),
	}
	if group != nil {
		switch schedulerType {
		case fairShareScheduling:
			job.Weight = group.weight
		case priorityScheduling:
			job.Priority = int32(*group.priority)
		}
	}

	return job
}

func mergeToJobQInfo(reqs AllocReqs) (map[model.JobID]*job.RMJobInfo, map[model.JobID]*actor.Ref) {
	isAdded := make(map[model.JobID]*job.RMJobInfo)
	jobActors := make(map[model.JobID]*actor.Ref)
	jobsAhead := 0
	for _, req := range reqs {
		curJob := req.Job
		if curJob == nil {
			continue
		}
		v1JobInfo, exists := isAdded[curJob.JobID]
		if !exists {
			v1JobInfo = &job.RMJobInfo{
				JobsAhead:      jobsAhead,
				State:          req.Job.State,
				RequestedSlots: req.Job.RequestedSlots,
				AllocatedSlots: req.Job.AllocatedSlots,
				IsPreemptible:  req.Preemptible,
			}
			isAdded[curJob.JobID] = v1JobInfo
			jobsAhead++
			jobActors[curJob.JobID] = req.Group
		}
		// Carry over the the highest state.
		if v1JobInfo.State < curJob.State {
			isAdded[curJob.JobID].State = curJob.State
		}
		v1JobInfo.RequestedSlots += req.SlotsNeeded
		if sproto.ScheduledStates[req.Job.State] {
			v1JobInfo.AllocatedSlots += req.SlotsNeeded
		}
	}
	return isAdded, jobActors
}

func addAllocateReqSlots(v1Job *jobv1.Job, req *sproto.AllocateRequest) {
	v1Job.RequestedSlots += int32(req.SlotsNeeded)
	if sproto.ScheduledStates[req.Job.State] {
		v1Job.AllocatedSlots += int32(req.SlotsNeeded)
	}
}

/* mergeToJobs
1. filters allocations that are not associated with a job
2. merges multilpe allocations representing a single job picking up information from all of them.
Input:
reqs: a list of allocateRequests sorted by expected order of execution from the scheduler.
extended: whether the costlier job attributes should be filled or not.
*/
func mergeToJobs( // TODO deprecate (k8)
	reqs AllocReqs,
	groups map[*actor.Ref]*group,
	schedulerType string,
) []*jobv1.Job {
	isAdded := make(map[model.JobID]*jobv1.Job)
	v1Jobs := make([]*jobv1.Job, 0)
	jobsAhead := 0
	for _, req := range reqs {
		curJob := req.Job
		if curJob == nil {
			continue
		}
		v1Job, exists := isAdded[curJob.JobID]
		if !exists {
			v1Job = allocateReqToV1Job(groups[req.Group], schedulerType, req, jobsAhead)
			isAdded[curJob.JobID] = v1Job
			v1Jobs = append(v1Jobs, v1Job)
			jobsAhead++
		}
		// Carry over the the highest state.
		if v1Job.Summary.State < curJob.State.Proto() {
			isAdded[curJob.JobID].Summary.State = curJob.State.Proto()
		}
		addAllocateReqSlots(v1Job, req)
	}

	return v1Jobs
}

// allocReqsToJobOrder converts sorted allocation requests to job order.
// func allocReqsToJobOrder(reqs []*sproto.AllocateRequest) (jobIds []string) {
// 	for _, job := range mergeToJobs(reqs, nil, DefaultSchedulerConfig().GetType()) {
// 		jobIds = append(jobIds, job.JobId)
// 	}
// 	return jobIds
// }

// order map keys by jobsAhead attribute of values
// func orderMapKeysByJobsAhead(m map[model.JobID]*job.RMJobInfo) (keys []string) {
// }

func setJobState(req *sproto.AllocateRequest, state sproto.SchedulingState) {
	if req.Job == nil {
		return
	}
	req.Job.State = state
}

func jobStats(rp *ResourcePool) *jobv1.QueueStats {
	stats := &jobv1.QueueStats{}
	jobinfo := rp.scheduler.JobQInfo(rp)
	for _, j := range jobinfo {
		if j.IsPreemptible {
			stats.PreemptibleCount++
		}
		if j.State == sproto.SchedulingStateQueued {
			stats.QueuedCount++
		} else {
			stats.ScheduledCount++
		}
	}
	return stats
}
