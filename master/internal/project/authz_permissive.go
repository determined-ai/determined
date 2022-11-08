package project

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// ProjectAuthZPermissive is the permission implementation.
type ProjectAuthZPermissive struct{}

// CanGetProject calls RBAC authz but enforces basic authz.
func (p *ProjectAuthZPermissive) CanGetProject(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) (bool, error) {
	_, _ = (&ProjectAuthZRBAC{}).CanGetProject(ctx, curUser, project)
	return (&ProjectAuthZBasic{}).CanGetProject(ctx, curUser, project)
}

// CanCreateProject calls RBAC authz but enforces basic authz.
func (p *ProjectAuthZPermissive) CanCreateProject(
	ctx context.Context, curUser model.User,
	willBeInWorkspace *workspacev1.Workspace,
) error {
	_ = (&ProjectAuthZRBAC{}).CanCreateProject(ctx, curUser, willBeInWorkspace)
	return (&ProjectAuthZBasic{}).CanCreateProject(ctx, curUser, willBeInWorkspace)
}

// CanSetProjectNotes calls RBAC authz but enforces basic authz.
func (p *ProjectAuthZPermissive) CanSetProjectNotes(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) error {
	_ = (&ProjectAuthZRBAC{}).CanSetProjectNotes(ctx, curUser, project)
	return (&ProjectAuthZBasic{}).CanSetProjectNotes(ctx, curUser, project)
}

// CanSetProjectName calls RBAC authz but enforces basic authz.
func (p *ProjectAuthZPermissive) CanSetProjectName(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) error {
	_ = (&ProjectAuthZRBAC{}).CanSetProjectName(ctx, curUser, project)
	return (&ProjectAuthZBasic{}).CanSetProjectName(ctx, curUser, project)
}

// CanSetProjectDescription calls RBAC authz but enforces basic authz.
func (p *ProjectAuthZPermissive) CanSetProjectDescription(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) error {
	_ = (&ProjectAuthZRBAC{}).CanSetProjectDescription(ctx, curUser, project)
	return (&ProjectAuthZBasic{}).CanSetProjectDescription(ctx, curUser, project)
}

// CanDeleteProject calls RBAC authz but enforces basic authz.
func (p *ProjectAuthZPermissive) CanDeleteProject(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) error {
	_ = (&ProjectAuthZRBAC{}).CanDeleteProject(ctx, curUser, project)
	return (&ProjectAuthZBasic{}).CanDeleteProject(ctx, curUser, project)
}

// CanMoveProject calls RBAC authz but enforces basic authz.
func (p *ProjectAuthZPermissive) CanMoveProject(
	ctx context.Context, curUser model.User, project *projectv1.Project,
	from, to *workspacev1.Workspace,
) error {
	_ = (&ProjectAuthZRBAC{}).CanMoveProject(ctx, curUser, project, from, to)
	return (&ProjectAuthZBasic{}).CanMoveProject(ctx, curUser, project, from, to)
}

// CanMoveProjectExperiments calls RBAC authz but enforces basic authz.
func (p *ProjectAuthZPermissive) CanMoveProjectExperiments(
	ctx context.Context, curUser model.User, exp *model.Experiment,
	from, to *projectv1.Project,
) error {
	_ = (&ProjectAuthZRBAC{}).CanMoveProjectExperiments(ctx, curUser, exp, from, to)
	return (&ProjectAuthZBasic{}).CanMoveProjectExperiments(ctx, curUser, exp, from, to)
}

// CanArchiveProject calls RBAC authz but enforces basic authz.
func (p *ProjectAuthZPermissive) CanArchiveProject(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) error {
	_ = (&ProjectAuthZRBAC{}).CanArchiveProject(ctx, curUser, project)
	return (&ProjectAuthZBasic{}).CanArchiveProject(ctx, curUser, project)
}

// CanUnarchiveProject calls RBAC authz but enforces basic authz.
func (p *ProjectAuthZPermissive) CanUnarchiveProject(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) error {
	_ = (&ProjectAuthZRBAC{}).CanUnarchiveProject(ctx, curUser, project)
	return (&ProjectAuthZBasic{}).CanUnarchiveProject(ctx, curUser, project)
}

func init() {
	AuthZProvider.Register("permissive", &ProjectAuthZPermissive{})
}
