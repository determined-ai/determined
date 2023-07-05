package job

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// JobAuthZPermissive is permissive OSS controls.
type JobAuthZPermissive struct{}

// FilterJobs returns a list of jobs that the user can view.
func (a *JobAuthZPermissive) FilterJobs(
	ctx context.Context, curUser model.User, jobs []*jobv1.Job,
) ([]*jobv1.Job, error) {
	_, _ = (&JobAuthZRBAC{}).FilterJobs(ctx, curUser, jobs)
	return (&JobAuthZBasic{}).FilterJobs(ctx, curUser, jobs)
}

// CanControlJobQueue returns an error if the user is not authorized to manipulate the
// job queue.
func (a *JobAuthZPermissive) CanControlJobQueue(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	_, _ = (&JobAuthZRBAC{}).CanControlJobQueue(ctx, curUser)
	return (&JobAuthZBasic{}).CanControlJobQueue(ctx, curUser)
}

func init() {
	AuthZProvider.Register("permissive", &JobAuthZPermissive{})
}
