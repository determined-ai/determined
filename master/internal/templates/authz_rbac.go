package templates

import (
	"context"

	"github.com/determined-ai/determined/master/internal/rbac"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// TemplateAuthZRBAC is the RBAC implementation of the TemplateAuthZ interface.
type TemplateAuthZRBAC struct{}

// ViewableScopes implements the TemplateAuthZ interface.
func (a *TemplateAuthZRBAC) ViewableScopes(
	ctx context.Context, curUser *model.User, requestedScope model.AccessScopeID,
) (model.AccessScopeSet, error) {
	return rbac.PermittedScopes(ctx, *curUser, requestedScope,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_TEMPLATES)
}

// CanUpdateTemplate checks if the user can update templates.
func (a *TemplateAuthZRBAC) CanUpdateTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	return rbac.CheckForPermission(ctx, "template", curUser,
		&workspaceID, rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_TEMPLATES)
}

// CanViewTemplate checks if the user can view the template.
func (a *TemplateAuthZRBAC) CanViewTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	return rbac.CheckForPermission(ctx, "template", curUser,
		&workspaceID, rbacv1.PermissionType_PERMISSION_TYPE_VIEW_TEMPLATES)
}

// CanDeleteTemplate checks if the user can delete the template.
func (a *TemplateAuthZRBAC) CanDeleteTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	return rbac.CheckForPermission(ctx, "template", curUser,
		&workspaceID, rbacv1.PermissionType_PERMISSION_TYPE_DELETE_TEMPLATES)
}

// CanCreateTemplate checks if the user can create the template.
func (a *TemplateAuthZRBAC) CanCreateTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	return rbac.CheckForPermission(ctx, "template", curUser,
		&workspaceID, rbacv1.PermissionType_PERMISSION_TYPE_CREATE_TEMPLATES)
}

func init() {
	AuthZProvider.Register("rbac", &TemplateAuthZRBAC{})
}
