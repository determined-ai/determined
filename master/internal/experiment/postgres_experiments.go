package experiment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/checkpoints"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/model"
)

var emptyMetadata = []byte(`{}`)

// GetExperimentAndCheckCanDoActions fetches an experiment and performs auth checks.
func GetExperimentAndCheckCanDoActions(
	ctx context.Context,
	expID int,
	actions ...func(context.Context, model.User, *model.Experiment) error,
) (*model.Experiment, model.User, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, model.User{}, err
	}

	e, err := db.ExperimentByID(ctx, expID)
	expNotFound := api.NotFoundErrs("experiment", fmt.Sprint(expID), true)
	if errors.Is(err, db.ErrNotFound) {
		return nil, model.User{}, expNotFound
	} else if err != nil {
		return nil, model.User{}, err
	}

	if err = AuthZProvider.Get().CanGetExperiment(ctx, *curUser, e); err != nil {
		return nil, model.User{}, authz.SubIfUnauthorized(err, expNotFound)
	}

	for _, action := range actions {
		if err = action(ctx, *curUser, e); err != nil {
			return nil, model.User{}, status.Errorf(codes.PermissionDenied, err.Error())
		}
	}
	return e, *curUser, nil
}

// ExperimentCheckpointsToGCRaw returns a comma-separated string describing checkpoints
// that should be GCed according to the given GC policy parameters. If the delete parameter is true,
// the returned checkpoints are also marked as deleted in the database.
func ExperimentCheckpointsToGCRaw(
	ctx context.Context,
	id int,
	experimentBest, trialBest, trialLatest int,
) ([]uuid.UUID, error) {
	// The string for the CTEs that we need whether or not we're not deleting the results. The
	// "selected_checkpoints" table contains the checkpoints to return as rows, so that we can easily
	// set the corresponding checkpoints to deleted in a separate CTE if we're deleting.
	// In the query the order includes the id to prevent different rows from having the same rank,
	// which could cause more than the desired number of checkpoints to be left out of the result set.
	// Also, any rows with null validation values will sort to the end, thereby not affecting the ranks
	// of rows with non-null validations, and will be filtered out later.
	query := `
WITH const AS (
    SELECT config->'searcher'->>'metric' AS metric_name,
           (CASE
                WHEN coalesce((config->'searcher'->>'smaller_is_better')::boolean, true)
                THEN 1
                ELSE -1
            END) AS sign
    FROM experiments WHERE id = ?
), selected_checkpoints AS (
	SELECT c.uuid,
		rank() OVER (
			ORDER BY const.sign * (v.metrics->'validation_metrics'->>const.metric_name)::float8
			ASC NULLS LAST, v.id ASC
		) AS experiment_rank,
		rank() OVER (
			PARTITION BY v.trial_id
			ORDER BY const.sign * (v.metrics->'validation_metrics'->>const.metric_name)::float8
			ASC NULLS LAST, v.id ASC
		) AS trial_rank,
		rank() OVER (
			PARTITION BY v.trial_id
			ORDER BY (c.metadata->>'steps_completed')::int DESC
		) AS trial_order_rank,
		v.metrics->'validation_metrics'->>const.metric_name as val_metric
	FROM checkpoints_v2 c
	JOIN const ON true
	JOIN run_id_task_id ON c.task_id = run_id_task_id.task_id
    JOIN trials t ON run_id_task_id.run_id = t.id
	LEFT JOIN validations v ON v.total_batches = (c.metadata->>'steps_completed')::int AND
		v.trial_id = t.id
	WHERE c.report_time IS NOT NULL
		AND (SELECT COUNT(*) FROM trials t WHERE t.warm_start_checkpoint_id = c.id) = 0
		AND t.experiment_id = ?
)
SELECT sc.uuid AS ID
FROM selected_checkpoints sc
WHERE ((experiment_rank > ? AND trial_rank > ?) OR (val_metric IS NULL))
	AND trial_order_rank > ?;`

	var checkpointIDRows []struct {
		ID uuid.UUID
	}

	if err := db.Bun().NewRaw(query,
		id, id, experimentBest, trialBest, trialLatest).Scan(ctx, &checkpointIDRows); err != nil {
		return nil, fmt.Errorf("querying for checkpoints that can be deleted according to the GC policy: %w", err)
	}

	var checkpointIDs []uuid.UUID
	for _, cRow := range checkpointIDRows {
		checkpointIDs = append(checkpointIDs, cRow.ID)
	}

	registeredCheckpoints, err := checkpoints.GetRegisteredCheckpoints(ctx, checkpointIDs)
	if err != nil {
		return nil, err
	}
	var deleteCheckpoints []uuid.UUID
	for _, cUUID := range checkpointIDs {
		if _, ok := registeredCheckpoints[cUUID]; !ok { // not a model registry checkpoint
			deleteCheckpoints = append(deleteCheckpoints, cUUID)
		}
	}

	return deleteCheckpoints, nil
}
