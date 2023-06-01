package job

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// JobAuthZBasic is basic OSS controls.
type JobAuthZBasic struct{}

// FilterJobs returns a list of jobs that the user can view.
func (a *JobAuthZBasic) FilterJobs(
	ctx context.Context, curUser model.User, jobs []*jobv1.Job,
) ([]*jobv1.Job, error) {
	return jobs, nil
}

func init() {
	AuthZProvider.Register("basic", &JobAuthZBasic{})
}
