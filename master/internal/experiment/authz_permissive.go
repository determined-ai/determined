package experiment

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

// ExperimentAuthZPermissive is the permission implementation.
type ExperimentAuthZPermissive struct{}

// CanGetExperiment calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanGetExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) (bool, error) {
	_, _ = (&ExperimentAuthZRBAC{}).CanGetExperiment(ctx, curUser, e)
	return (&ExperimentAuthZBasic{}).CanGetExperiment(ctx, curUser, e)
}

// CanGetExperimentArtifacts calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanGetExperimentArtifacts(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanGetExperimentArtifacts(ctx, curUser, e)
	return (&ExperimentAuthZBasic{}).CanGetExperimentArtifacts(ctx, curUser, e)
}

// CanDeleteExperiment calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanDeleteExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanDeleteExperiment(ctx, curUser, e)
	return (&ExperimentAuthZBasic{}).CanDeleteExperiment(ctx, curUser, e)
}

// FilterExperimentsQuery calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) FilterExperimentsQuery(
	ctx context.Context, curUser model.User, proj *projectv1.Project,
	query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	_, _ = (&ExperimentAuthZRBAC{}).FilterExperimentsQuery(ctx, curUser, proj, query)
	return (&ExperimentAuthZBasic{}).FilterExperimentsQuery(ctx, curUser, proj, query)
}

// FilterExperimentLabelsQuery calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) FilterExperimentLabelsQuery(
	ctx context.Context, curUser model.User, proj *projectv1.Project,
	query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	_, _ = (&ExperimentAuthZRBAC{}).FilterExperimentLabelsQuery(ctx, curUser, proj, query)
	return (&ExperimentAuthZBasic{}).FilterExperimentLabelsQuery(ctx, curUser, proj, query)
}

// CanPreviewHPSearch calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanPreviewHPSearch(
	ctx context.Context, curUser model.User,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanPreviewHPSearch(ctx, curUser)
	return (&ExperimentAuthZBasic{}).CanPreviewHPSearch(ctx, curUser)
}

// CanEditExperiment calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanEditExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanEditExperiment(ctx, curUser, e)
	return (&ExperimentAuthZBasic{}).CanEditExperiment(ctx, curUser, e)
}

// CanEditExperimentsMetadata calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanEditExperimentsMetadata(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanEditExperimentsMetadata(ctx, curUser, e)
	return (&ExperimentAuthZBasic{}).CanEditExperimentsMetadata(ctx, curUser, e)
}

// CanCreateExperiment calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanCreateExperiment(
	ctx context.Context, curUser model.User, proj *projectv1.Project,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanCreateExperiment(ctx, curUser, proj)
	return (&ExperimentAuthZBasic{}).CanCreateExperiment(ctx, curUser, proj)
}

// CanForkFromExperiment calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanForkFromExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanForkFromExperiment(ctx, curUser, e)
	return (&ExperimentAuthZBasic{}).CanForkFromExperiment(ctx, curUser, e)
}

// CanSetExperimentsMaxSlots calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanSetExperimentsMaxSlots(
	ctx context.Context, curUser model.User, e *model.Experiment, slots int,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanSetExperimentsMaxSlots(ctx, curUser, e, slots)
	return (&ExperimentAuthZBasic{}).CanSetExperimentsMaxSlots(ctx, curUser, e, slots)
}

// CanSetExperimentsWeight calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanSetExperimentsWeight(
	ctx context.Context, curUser model.User, e *model.Experiment, weight float64,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanSetExperimentsWeight(ctx, curUser, e, weight)
	return (&ExperimentAuthZBasic{}).CanSetExperimentsWeight(ctx, curUser, e, weight)
}

// CanSetExperimentsPriority calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanSetExperimentsPriority(
	ctx context.Context, curUser model.User, e *model.Experiment, priority int,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanSetExperimentsPriority(ctx, curUser, e, priority)
	return (&ExperimentAuthZBasic{}).CanSetExperimentsPriority(ctx, curUser, e, priority)
}

// CanSetExperimentsCheckpointGCPolicy calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanSetExperimentsCheckpointGCPolicy(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanSetExperimentsCheckpointGCPolicy(ctx, curUser, e)
	return (&ExperimentAuthZBasic{}).CanSetExperimentsCheckpointGCPolicy(ctx, curUser, e)
}

// CanRunCustomSearch calls RBAC authz but enforces basic authz.
func (p *ExperimentAuthZPermissive) CanRunCustomSearch(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	_ = (&ExperimentAuthZRBAC{}).CanRunCustomSearch(ctx, curUser, e)
	return (&ExperimentAuthZBasic{}).CanRunCustomSearch(ctx, curUser, e)
}

func init() {
	AuthZProvider.Register("permissive", &ExperimentAuthZPermissive{})
}
