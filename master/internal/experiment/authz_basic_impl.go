package experiment

import (
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

// ExperimentAuthZBasic is basic OSS controls.
type ExperimentAuthZBasic struct{}

// CanGetExperiment always returns true and a nill error.
func (a *ExperimentAuthZBasic) CanGetExperiment(
	curUser model.User, e *model.Experiment,
) (canGetExp bool, serverError error) {
	return true, nil
}

// CanDeleteExperiment returns an error if the experiment
// is not owned by the current user and the current user is not an admin.
func (a *ExperimentAuthZBasic) CanDeleteExperiment(curUser model.User, e *model.Experiment) error {
	curUserIsOwner := e.OwnerID == nil || *e.OwnerID == curUser.ID
	if !curUser.Admin && !curUserIsOwner {
		return fmt.Errorf("non admin users may not delete other user's experiments")
	}
	return nil
}

// FilterExperimentsQuery returns the query unmodified and a nil error.
func (a *ExperimentAuthZBasic) FilterExperimentsQuery(
	curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	return query, nil
}

// FilterExperimentLabelsQuery returns the query unmodified and a nil error.
func (a *ExperimentAuthZBasic) FilterExperimentLabelsQuery(
	curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	return query, nil
}

// CanGetExperimentValidationHistory always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetExperimentValidationHistory(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanPreviewHPSearch always returns a nil error.
func (a *ExperimentAuthZBasic) CanPreviewHPSearch(curUser model.User) error {
	return nil
}

// CanActivateExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanActivateExperiment(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanPauseExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanPauseExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanCancelExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanCancelExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanKillExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanKillExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanArchiveExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanArchiveExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanUnarchiveExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanUnarchiveExperiment(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanSetExperimentsName always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsName(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanSetExperimentsNotes always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsNotes(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanSetExperimentsDescription always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsDescription(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanSetExperimentsLabels always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsLabels(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

// FilterCheckpoints returns the input list and a nil error.
func (a *ExperimentAuthZBasic) FilterCheckpoints(
	curUser model.User, e *model.Experiment, checkpoints []*checkpointv1.Checkpoint,
) ([]*checkpointv1.Checkpoint, error) {
	return checkpoints, nil
}

// CanCreateExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanCreateExperiment(
	curUser model.User, proj *projectv1.Project, e *model.Experiment,
) error {
	return nil
}

// CanForkFromExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanForkFromExperiment(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanGetMetricNames always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetMetricNames(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanGetMetricBatches always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetMetricBatches(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanGetTrialsSnapshot always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetTrialsSnapshot(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanGetTrialsSample always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetTrialsSample(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanComputeHPImportance always returns a nil error.
func (a *ExperimentAuthZBasic) CanComputeHPImportance(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanGetHPImportance always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetHPImportance(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanGetBestSearcherValidationMetric always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetBestSearcherValidationMetric(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanGetModelDef always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetModelDef(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanMoveExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanMoveExperiment(
	curUser model.User, from, to *projectv1.Project, e *model.Experiment,
) error {
	return nil
}

// CanGetModelDefTree always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetModelDefTree(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanGetModelDefFile always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetModelDefFile(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanGetExperimentsCheckpointsToGC always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetExperimentsCheckpointsToGC(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanSetExperimentsMaxSlots always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsMaxSlots(
	curUser model.User, e *model.Experiment, slots int,
) error {
	return nil
}

// CanSetExperimentsWeight always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsWeight(
	curUser model.User, e *model.Experiment, weight float64,
) error {
	return nil
}

// CanSetExperimentsPriority always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsPriority(
	curUser model.User, e *model.Experiment, priority int,
) error {
	return nil
}

// CanSetExperimentsCheckpointGCPolicy always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsCheckpointGCPolicy(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &ExperimentAuthZBasic{})
}
