package experiment

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// ExperimentAuthZ describes authz methods for experiments.
type ExperimentAuthZ interface {
	// GET /api/v1/experiments/:exp_id
	// GET /tasks
	CanGetExperiment(
		ctx context.Context, curUser model.User, e *model.Experiment,
	) error

	// GET /api/v1/experiments/:exp_id/file_tree
	// POST /api/v1/experiments/{experimentId}/file
	// GET /experiments/:exp_id/file/download
	// GET /api/v1/experiments/:exp_id/model_def
	// GET /experiments/:exp_id/model_def
	// GET /api/v1/experiments/:exp_id/checkpoints
	// GET /experiments/:exp_id/preview_gc
	// GET /api/v1/experiments/:exp_id/validation_history
	// GET /api/v1/experiments/:exp_id/searcher/best_searcher_validation_metric
	// GET /api/v1/experiments/:exp_id/metrics-stream/metric-names
	// GET /api/v1/experiments/:exp_id/metrics-stream/batches
	// GET /api/v1/experiments/:exp_id/metrics-stream/trials-snapshot
	// GET /api/v1/experiments/:exp_id/metrics-stream/trials-sample
	// GET /api/v1/experiments/{experimentId}/hyperparameter-importance
	// GET /api/v1/trials/:trial_id/checkpoints
	// GET /api/v1/experiments/:trial_id/trials
	// GET /api/v1/trials/:trial_id
	// GET /api/v1/trials/:trial_id/summarize
	// GET /api/v1/trials/compare
	// GET /api/v1/trials/:trial_id/workloads
	// GET /api/v1/trials/:trial_id/profiler/metrics
	// GET /api/v1/trials/:trial_id/profiler/available_series
	// GET /api/v1/trials/:trial_id/searcher/operation
	// GET /api/v1/trials/:trial_id/logs
	// GET /api/v1/trials/:trial_id/logs/fields
	// GET /trials/:trial_id
	// GET /trials/:trial_id/metrics
	CanGetExperimentArtifacts(ctx context.Context, curUser model.User, e *model.Experiment) error

	// DELETE /api/v1/experiments/:exp_id
	CanDeleteExperiment(ctx context.Context, curUser model.User, e *model.Experiment) error

	// GET /api/v1/experiments
	// "proj" being nil indicates getting experiments from all projects.
	// WARN: query is expected to expose the "workspace_id" column.
	FilterExperimentsQuery(
		ctx context.Context, curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
		permissions []rbacv1.PermissionType,
	) (*bun.SelectQuery, error)

	// GET /api/v1/experiments/labels
	// "proj" being nil indicates searching across all projects.
	FilterExperimentLabelsQuery(
		ctx context.Context, curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
	) (*bun.SelectQuery, error)

	// POST /api/v1/preview-hp-search
	CanPreviewHPSearch(ctx context.Context, curUser model.User) error

	// POST /api/v1/experiments/:exp_id/activate
	// POST /api/v1/experiments
	// POST /api/v1/experiments/:exp_id/pause
	// POST /api/v1/experiments/:exp_id/kill
	// POST /api/v1/experiments/:exp_id/hyperparameter-importance
	// POST /api/v1/experiments/:exp_id/cancel
	// POST /api/v1/trials/:trial_id/kill
	// POST /api/v1/trials/profiler/metrics
	// POST /api/v1/trials/:trial_id/searcher/completed_operation
	// POST /api/v1/trials/:trial_id/early_exit
	// POST /api/v1/trials/:trial_id/progress
	// POST /api/v1/trials/:trial_id/training_metrics
	// POST /api/v1/trials/:trial_id/validation_metrics
	// POST /api/v1/trials/:trial_id/runner/metadata
	// POST /api/v1/allocations/:allocation_id/all_gather
	// POST /api/v1/allocations/:allocation_id/proxy_address
	// POST /api/v1/allocations/:allocation_id/waiting
	CanEditExperiment(ctx context.Context, curUser model.User, e *model.Experiment) error

	// POST /api/v1/experiments/:exp_id/archive
	// POST /api/v1/experiments/:exp_id/unarchive
	// PATCH /api/v1/experiments/:exp_id/
	CanEditExperimentsMetadata(ctx context.Context, curUser model.User, e *model.Experiment) error

	// POST /api/v1/experiments
	CanCreateExperiment(
		ctx context.Context, curUser model.User, proj *projectv1.Project,
	) error
	CanForkFromExperiment(ctx context.Context, curUser model.User, e *model.Experiment) error

	// PATCH /experiments/:exp_id
	CanSetExperimentsMaxSlots(
		ctx context.Context, curUser model.User, e *model.Experiment, slots int,
	) error
	CanSetExperimentsWeight(
		ctx context.Context, curUser model.User, e *model.Experiment, weight float64,
	) error
	CanSetExperimentsPriority(
		ctx context.Context, curUser model.User, e *model.Experiment, priority int,
	) error
	CanSetExperimentsCheckpointGCPolicy(
		ctx context.Context, curUser model.User, e *model.Experiment,
	) error

	// GET /api/v1/experiments/:exp_id/searcher_events
	// POST /api/v1/experiments/:exp_id/searcher_operations
	CanRunCustomSearch(ctx context.Context, curUser model.User, e *model.Experiment) error
}

// AuthZProvider is the authz registry for experiments.
var AuthZProvider authz.AuthZProviderType[ExperimentAuthZ]
