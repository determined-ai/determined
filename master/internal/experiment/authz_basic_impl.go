package experiment

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// ExperimentAuthZBasic is basic OSS controls.
type ExperimentAuthZBasic struct{}

// CanGetExperiment always returns true and a nill error.
func (a *ExperimentAuthZBasic) CanGetExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanGetExperimentArtifacts always returns a nil error.
func (a *ExperimentAuthZBasic) CanGetExperimentArtifacts(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanDeleteExperiment returns an error if the experiment
// is not owned by the current user and the current user is not an admin.
func (a *ExperimentAuthZBasic) CanDeleteExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	curUserIsOwner := e.OwnerID == nil || *e.OwnerID == curUser.ID
	if !curUser.Admin && !curUserIsOwner {
		return fmt.Errorf("non admin users may not delete other user's experiments")
	}
	return nil
}

// FilterExperimentsQuery returns the query unmodified and a nil error.
func (a *ExperimentAuthZBasic) FilterExperimentsQuery(
	ctx context.Context, curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
	permissions []rbacv1.PermissionType,
) (*bun.SelectQuery, error) {
	return query, nil
}

// FilterExperimentLabelsQuery returns the query unmodified and a nil error.
func (a *ExperimentAuthZBasic) FilterExperimentLabelsQuery(
	ctx context.Context, curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	return query, nil
}

// CanPreviewHPSearch always returns a nil error.
func (a *ExperimentAuthZBasic) CanPreviewHPSearch(
	ctx context.Context, curUser model.User,
) error {
	return nil
}

// CanEditExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanEditExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanEditExperimentsMetadata always returns a nil error.
func (a *ExperimentAuthZBasic) CanEditExperimentsMetadata(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanCreateExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanCreateExperiment(
	ctx context.Context, curUser model.User, proj *projectv1.Project,
) error {
	return nil
}

// CanForkFromExperiment always returns a nil error.
func (a *ExperimentAuthZBasic) CanForkFromExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanSetExperimentsMaxSlots always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsMaxSlots(
	ctx context.Context, curUser model.User, e *model.Experiment, slots int,
) error {
	return nil
}

// CanSetExperimentsWeight always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsWeight(
	ctx context.Context, curUser model.User, e *model.Experiment, weight float64,
) error {
	return nil
}

// CanSetExperimentsPriority always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsPriority(
	ctx context.Context, curUser model.User, e *model.Experiment, priority int,
) error {
	return nil
}

// CanSetExperimentsCheckpointGCPolicy always returns a nil error.
func (a *ExperimentAuthZBasic) CanSetExperimentsCheckpointGCPolicy(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	return nil
}

// CanRunCustomSearch always returns a nil error.
func (a *ExperimentAuthZBasic) CanRunCustomSearch(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &ExperimentAuthZBasic{})
}
