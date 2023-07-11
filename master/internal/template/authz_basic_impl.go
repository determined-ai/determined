package template

import (
	"context"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TemplateAuthZBasic is basic OSS controls.
type TemplateAuthZBasic struct{}

// ViewableScopes implements the TemplateAuthZ interface.
func (a *TemplateAuthZBasic) ViewableScopes(
	ctx context.Context, curUser *model.User, requestedScope model.AccessScopeID,
) (model.AccessScopeSet, error) {
	var ids []int
	returnScope := model.AccessScopeSet{requestedScope: true}

	if requestedScope == 0 {
		err := db.Bun().NewSelect().Table("workspaces").Column("id").Scan(ctx, &ids)
		if err != nil {
			return nil, err
		}

		for _, id := range ids {
			returnScope[model.AccessScopeID(id)] = true
		}

		return returnScope, nil
	}
	return returnScope, nil
}

// CanCreateTemplate implements the TemplateAuthZ interface.
func (a *TemplateAuthZBasic) CanCreateTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	return nil, nil
}

// CanViewTemplate implements the TemplateAuthZ interface.
func (a *TemplateAuthZBasic) CanViewTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	return nil, nil
}

// CanUpdateTemplate implements the TemplateAuthZ interface.
func (a *TemplateAuthZBasic) CanUpdateTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	return nil, nil
}

// CanDeleteTemplate implements the TemplateAuthZ interface.
func (a *TemplateAuthZBasic) CanDeleteTemplate(
	ctx context.Context, curUser *model.User, workspaceID model.AccessScopeID,
) (permErr error, err error) {
	return nil, nil
}

func init() {
	AuthZProvider.Register("basic", &TemplateAuthZBasic{})
}
