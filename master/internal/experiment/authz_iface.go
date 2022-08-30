package experiment

import (
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

// ExperimentAuthZ describes authz methods for experiments.
type ExperimentAuthZ interface {
	// GET /api/v1/experiments/:exp_id
	CanGetExperiment(curUser model.User, e *model.Experiment) (canGetExp bool, serverError error)

	// DELETE /api/v1/experiments/:exp_id
	CanDeleteExperiment(curUser model.User, e *model.Experiment) error

	// GET /api/v1/experiments
	// "proj" being nil indicates getting experiments from all projects.
	FilterExperimentsQuery(
		curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
	) (*bun.SelectQuery, error)

	// GET /api/v1/experiments/labels
	// "proj" being nil indicates searching across all projects.
	FilterExperimentLabelsQuery(
		curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
	) (*bun.SelectQuery, error)

	// GET /api/v1/experiments/:exp_id/validation_history
	CanGetExperimentValidationHistory(curUser model.User, e *model.Experiment) error

	// POST /api/v1/preview-hp-search
	CanPreviewHPSearch(curUser model.User) error

	// POST /api/v1/experiments/:exp_id/activate
	// POST /api/v1/experiments
	CanActivateExperiment(curUser model.User, e *model.Experiment) error

	// POST /api/v1/experiments/:exp_id/pause
	CanPauseExperiment(curUser model.User, e *model.Experiment) error
	// POST /api/v1/experiments/:exp_id/cancel
	CanCancelExperiment(curUser model.User, e *model.Experiment) error
	// POST /api/v1/experiments/:exp_id/kill
	CanKillExperiment(curUser model.User, e *model.Experiment) error

	// POST /api/v1/experiments/:exp_id/archive
	CanArchiveExperiment(curUser model.User, e *model.Experiment) error
	// POST /api/v1/experiments/:exp_id/unarchive
	CanUnarchiveExperiment(curUser model.User, e *model.Experiment) error

	// PATCH /api/v1/experiments/:exp_id/
	CanSetExperimentsName(curUser model.User, e *model.Experiment) error
	CanSetExperimentsNotes(curUser model.User, e *model.Experiment) error
	CanSetExperimentsDescription(curUser model.User, e *model.Experiment) error
	CanSetExperimentsLabels(curUser model.User, e *model.Experiment) error

	// GET /api/v1/experiments/:exp_id/checkpoints
	CanGetExperimentsCheckpoints(curUser model.User, e *model.Experiment) error

	// POST /api/v1/experiments
	CanCreateExperiment(curUser model.User, proj *projectv1.Project, e *model.Experiment) error
	CanForkFromExperiment(curUser model.User, e *model.Experiment) error

	// GET /api/v1/experiments/:exp_id/metrics-stream/metric-names
	CanGetMetricNames(curUser model.User, e *model.Experiment) error

	// GET /api/v1/experiments/:exp_id/metrics-stream/batches
	CanGetMetricBatches(curUser model.User, e *model.Experiment) error

	// GET /api/v1/experiments/:exp_id/metrics-stream/trials-snapshot
	CanGetTrialsSnapshot(curUser model.User, e *model.Experiment) error

	// GET /api/v1/experiments/:exp_id/metrics-stream/trials-sample
	CanGetTrialsSample(curUser model.User, e *model.Experiment) error

	// POST /api/v1/experiments/:exp_id/hyperparameter-importance
	CanComputeHPImportance(curUser model.User, e *model.Experiment) error

	// GET /api/v1/experiments/{experimentId}/hyperparameter-importance
	CanGetHPImportance(curUser model.User, e *model.Experiment) error

	// GET /api/v1/experiments/:exp_id/searcher/best_searcher_validation_metric
	CanGetBestSearcherValidationMetric(curUser model.User, e *model.Experiment) error

	// GET /api/v1/experiments/:exp_id/model_def
	// GET /experiments/:exp_id/model_def
	CanGetModelDef(curUser model.User, e *model.Experiment) error

	// POST /api/v1/experiments/:exp_id/move
	CanMoveExperiment(curUser model.User, from, to *projectv1.Project, e *model.Experiment) error

	// GET /api/v1/experiments/:exp_id/file_tree
	CanGetModelDefTree(curUser model.User, e *model.Experiment) error

	// POST /api/v1/experiments/{experimentId}/file
	// GET /experiments/:exp_id/file/download
	CanGetModelDefFile(curUser model.User, e *model.Experiment) error

	// GET /experiments/:exp_id/preview_gc
	CanGetExperimentsCheckpointsToGC(curUser model.User, e *model.Experiment) error

	// PATCH /experiments/:exp_id
	CanSetExperimentsMaxSlots(curUser model.User, e *model.Experiment, slots int) error
	CanSetExperimentsWeight(curUser model.User, e *model.Experiment, weight float64) error
	CanSetExperimentsPriority(curUser model.User, e *model.Experiment, priority int) error
	CanSetExperimentsCheckpointGCPolicy(curUser model.User, e *model.Experiment) error
}

// AuthZProvider is the authz registry for experiments.
var AuthZProvider authz.AuthZProviderType[ExperimentAuthZ]
