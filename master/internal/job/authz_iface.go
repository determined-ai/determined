package job

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// JobAuthZ describes authz methods for jobs.
type JobAuthZ interface {
	// FilterJobs returns a list of jobs that the user is authorized to view.
	FilterJobs(
		ctx context.Context, curUser model.User, jobs []*jobv1.Job,
	) ([]*jobv1.Job, error)
}

// AuthZProvider is the authz registry for Notebooks, Shells, and Commands.
var AuthZProvider authz.AuthZProviderType[JobAuthZ]
