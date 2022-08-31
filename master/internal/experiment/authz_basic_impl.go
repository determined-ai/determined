package experiment

import (
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
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

// CanGetExperimentArtifacts always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetExperimentArtifacts(
	curUser model.User, e *model.Experiment,
) error {
	return nil
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

// CanPreviewHPSearch always returns a nil error.
func (a *ExperimentAuthZBasic) CanPreviewHPSearch(curUser model.User) error {
	return nil
}

// CanEditExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanEditExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

// CanEditExperimentsMetadata always returns a nil error.
func (a *ExperimentAuthZBasic) CanEditExperimentsMetadata(
	curUser model.User, e *model.Experiment,
) error {
	return nil
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
