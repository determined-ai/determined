package templates

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TemplateAuthZPermissive is permissive implementation of the TemplateAuthZ interface.
type TemplateAuthZPermissive struct{}

// ViewableScopes logs the request.
func (a *TemplateAuthZPermissive) ViewableScopes(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (model.AccessScopeSet, error) {
	_, _ = (&TemplateAuthZRBAC{}).ViewableScopes(ctx, curUser, workspaceID)
	return (&TemplateAuthZBasic{}).ViewableScopes(ctx, curUser, workspaceID)
}

// CanCreateTemplate logs the request.
func (a *TemplateAuthZPermissive) CanCreateTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	_, _ = (&TemplateAuthZRBAC{}).CanCreateTemplate(ctx, curUser, workspaceID)
	return (&TemplateAuthZBasic{}).CanCreateTemplate(ctx, curUser, workspaceID)
}

// CanViewTemplate logs the request.
func (a *TemplateAuthZPermissive) CanViewTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	_, _ = (&TemplateAuthZRBAC{}).CanViewTemplate(ctx, curUser, workspaceID)
	return (&TemplateAuthZBasic{}).CanViewTemplate(ctx, curUser, workspaceID)
}

// CanUpdateTemplate logs the request.
func (a *TemplateAuthZPermissive) CanUpdateTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	_, _ = (&TemplateAuthZRBAC{}).CanUpdateTemplate(ctx, curUser, workspaceID)
	return (&TemplateAuthZBasic{}).CanUpdateTemplate(ctx, curUser, workspaceID)
}

// CanDeleteTemplate logs the request.
func (a *TemplateAuthZPermissive) CanDeleteTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	_, _ = (&TemplateAuthZRBAC{}).CanDeleteTemplate(ctx, curUser, workspaceID)
	return (&TemplateAuthZBasic{}).CanDeleteTemplate(ctx, curUser, workspaceID)
}

func init() {
	AuthZProvider.Register("permissive", &TemplateAuthZPermissive{})
}
