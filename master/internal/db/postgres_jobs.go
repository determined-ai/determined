package db

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// AddJobTx persists the existence of a job with a transaction.
func AddJobTx(ctx context.Context, idb bun.IDB, j *model.Job) error {
	if idb == nil {
		idb = Bun()
	}

	if _, err := idb.NewInsert().Model(j).Exec(ctx); err != nil {
		return fmt.Errorf("adding job: %w", err)
	}

	return nil
}

// AddJob persists the existence of a job.
func AddJob(j *model.Job) error {
	return AddJobTx(context.TODO(), Bun(), j)
}

// JobByID retrieves a job by ID.
func JobByID(ctx context.Context, jobID model.JobID) (*model.Job, error) {
	var j model.Job
	err := Bun().NewSelect().Model(&j).
		Where("job_id = ?", jobID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying job: %w", err)
	}
	return &j, nil
}
