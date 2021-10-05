package model

// jobQueue maintains tasks in the job queue order
type JobQueue struct {
	//  heap data structure?
	jobs    []*Job
	jobsMap map[JobID]*Job
	len     int
}

func (jq JobQueue) Len() int {
	return jq.len
}

func (jq JobQueue) Less(i, j int) bool {
	return jq.jobs[i].QPos < jq.jobs[j].QPos
}

func (jq JobQueue) Update(job *Job) {

}
