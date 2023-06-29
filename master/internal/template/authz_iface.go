package template

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TemplateAuthZ describes authz methods for template actions.
type TemplateAuthZ interface {
	// ViewableScopes returns the set of scopes that the user can view templates in.
	ViewableScopes(
		ctx context.Context, curUser *model.User, requestedScope model.AccessScopeID,
	) (model.AccessScopeSet, error)

	// CanCreateTemplate checks if the user can create a template.
	CanCreateTemplate(
		ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
	) (permErr error, err error)

	// CanViewTemplate checks if the user can view a template.
	CanViewTemplate(
		ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
	) (permErr error, err error)

	// CanUpdateTemplate checks if the user can update a template.
	CanUpdateTemplate(
		ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
	) (permErr error, err error)

	// CanDeleteTemplate checks if the user can delete a template.
	CanDeleteTemplate(
		ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
	) (permErr error, err error)
}

// AuthZProvider is the authz registry for Notebooks, Shells, and Commands.
var AuthZProvider authz.AuthZProviderType[TemplateAuthZ]
