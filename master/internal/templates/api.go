package templates

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/templatev1"
)

// TemplateAPIServer implements the template APIs for Determined's API server.
type TemplateAPIServer struct{}

// GetTemplates viewable by the user. If there are no matches, returns an empty list.
func (a *TemplateAPIServer) GetTemplates(
	ctx context.Context,
	req *apiv1.GetTemplatesRequest,
) (*apiv1.GetTemplatesResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	scopes, err := AuthZProvider.Get().ViewableScopes(ctx, user, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to check for permissions: %w", err)
	}

	var resp apiv1.GetTemplatesResponse
	err = db.Bun().NewSelect().Table("templates").ColumnExpr("*").Scan(ctx, &resp.Templates)
	if err != nil {
		return nil, fmt.Errorf("fetching templates from database: %w", err)
	}
	api.Where(&resp.Templates, func(i int) bool {
		if !scopes[model.AccessScopeID(resp.Templates[i].WorkspaceId)] {
			return false
		}
		return strings.Contains(strings.ToLower(resp.Templates[i].Name), strings.ToLower(req.Name))
	})
	api.Sort(resp.Templates, req.OrderBy, req.SortBy, apiv1.GetTemplatesRequest_SORT_BY_NAME)
	return &resp, api.Paginate(&resp.Pagination, &resp.Templates, req.Offset, req.Limit)
}

// GetTemplate by name. Returns an error if the requested template does not exist.
func (a *TemplateAPIServer) GetTemplate(
	ctx context.Context,
	req *apiv1.GetTemplateRequest,
) (*apiv1.GetTemplateResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.TemplateName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	var t templatev1.Template
	err = db.Bun().NewSelect().Table("templates").Where("name = ?", req.TemplateName).Scan(ctx, &t)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, api.NotFoundErrs("template", req.TemplateName, true)
	case err != nil:
		return nil, fmt.Errorf("fetching template %s from database: %w", req.TemplateName, err)
	}

	permErr, err := AuthZProvider.Get().CanViewTemplate(ctx, user, model.AccessScopeID(t.WorkspaceId))
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to check for permissions: %w", err)
	case permErr != nil:
		return nil, authz.SubIfUnauthorized(permErr, api.NotFoundErrs("template", req.TemplateName, true))
	}
	return &apiv1.GetTemplateResponse{Template: &t}, nil
}

// PutTemplate creates or updates a template.
func (a *TemplateAPIServer) PutTemplate(
	ctx context.Context,
	req *apiv1.PutTemplateRequest,
) (*apiv1.PutTemplateResponse, error) {
	if len(req.Template.Name) == 0 {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Template.WorkspaceId != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "setting workspace_id is not supported.")
	}

	switch _, err := TemplateByName(ctx, req.Template.Name); {
	case errors.Is(err, db.ErrNotFound):
		_, err := a.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: req.Template})
		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		_, err = a.PatchTemplateConfig(
			ctx,
			&apiv1.PatchTemplateConfigRequest{
				TemplateName: req.Template.Name,
				Config:       req.Template.Config,
			},
		)
		if err != nil {
			return nil, err
		}
	}
	return &apiv1.PutTemplateResponse{Template: req.Template}, nil
}

// PostTemplate creates a template. If a template with the same name exists, an error is returned.
func (a *TemplateAPIServer) PostTemplate(
	ctx context.Context,
	req *apiv1.PostTemplateRequest,
) (*apiv1.PostTemplateResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.Template.Name) == 0 {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	workspaceID := int(req.Template.WorkspaceId)
	if req.Template.WorkspaceId == 0 {
		workspaceID = model.DefaultWorkspaceID
	}

	err = workspace.AuthZProvider.Get().CanGetWorkspaceID(ctx, *user, req.Template.WorkspaceId)
	if err != nil {
		return nil, err
	}

	exists, err := workspace.Exists(ctx, workspaceID)
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to check workspace %d: %w", workspaceID, err)
	case !exists:
		return nil, api.NotFoundErrs("workspace", fmt.Sprint(workspaceID), true)
	}

	permErr, err := AuthZProvider.Get().CanCreateTemplate(
		ctx,
		user,
		model.AccessScopeID(workspaceID),
	)
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to check for permissions: %w", err)
	case permErr != nil:
		return nil, permErr
	}

	var inserted templatev1.Template
	err = db.Bun().NewInsert().Model(&model.Template{}).
		Column("name", "config", "workspace_id").
		Value("name", "?", req.Template.Name).
		Value("config", "?", req.Template.Config.AsMap()).
		Value("workspace_id", "?", workspaceID).
		Returning("name, config, workspace_id").
		Scan(ctx, &inserted)
	if err != nil {
		return nil, fmt.Errorf("failed to create template %s: %w", req.Template.Name, err)
	}
	return &apiv1.PostTemplateResponse{Template: &inserted}, nil
}

// PatchTemplateConfig does a full update of the requested template's config.
func (a *TemplateAPIServer) PatchTemplateConfig(
	ctx context.Context,
	req *apiv1.PatchTemplateConfigRequest,
) (*apiv1.PatchTemplateConfigResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.TemplateName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	tpl, err := TemplateByName(ctx, req.TemplateName)
	if err != nil {
		return nil, err
	}
	permErr, err := AuthZProvider.Get().CanUpdateTemplate(
		ctx, user, model.AccessScopeID(tpl.WorkspaceID),
	)
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to check for permissions: %w", err)
	case permErr != nil:
		return nil, permErr
	}

	var updated templatev1.Template
	err = db.Bun().NewUpdate().Model(&model.Template{}).
		Column("config").
		Set("config = ?", req.Config.AsMap()).
		Where("name = ?", req.TemplateName).
		Returning("name, config, workspace_id").
		Scan(ctx, &updated)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, api.NotFoundErrs("template", req.TemplateName, true)
	case err != nil:
		return nil, fmt.Errorf("failed to update template: %w", err)
	}
	return &apiv1.PatchTemplateConfigResponse{Template: &updated}, nil
}

// DeleteTemplate a template by name. Returns an error if the template does not exist.
func (a *TemplateAPIServer) DeleteTemplate(
	ctx context.Context, req *apiv1.DeleteTemplateRequest,
) (*apiv1.DeleteTemplateResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.TemplateName) == 0 {
		return nil, errors.New("error deleting template: empty name")
	}

	tpl, err := TemplateByName(ctx, req.TemplateName)
	if err != nil {
		return nil, err
	}
	permErr, err := AuthZProvider.Get().CanDeleteTemplate(
		ctx, user, model.AccessScopeID(tpl.WorkspaceID),
	)
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to check for permissions: %w", err)
	case permErr != nil:
		return nil, permErr
	}

	_, err = db.Bun().NewDelete().Table("templates").Where("name = ?", req.TemplateName).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("error deleting template '%v': %w", req.TemplateName, err)
	}
	return &apiv1.DeleteTemplateResponse{}, nil
}
