package templates

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/db/bunutils"
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

	q := db.Bun().NewSelect().
		Table("templates").
		ColumnExpr("*").
		Where("workspace_id IN (?)", bun.In(maps.Keys(scopes)))
	if req.Name != "" {
		q.Where("name ILIKE  ('%%' || ? || '%%')", req.Name)
	}
	q.Order(fmt.Sprintf("name %s", grpcutil.OrderBySQL[req.OrderBy])) // Only name is supported.
	q, pagination, err := bunutils.Paginate(ctx, q, int(req.Offset), int(req.Limit))
	if err != nil {
		return nil, fmt.Errorf("failed to paginate query: %w", err)
	}

	var tpls []*templatev1.Template
	err = q.Scan(ctx, &tpls)
	if err != nil {
		return nil, fmt.Errorf("fetching templates from database: %w", err)
	}

	return &apiv1.GetTemplatesResponse{Templates: tpls, Pagination: pagination}, nil
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

	var tpl templatev1.Template
	err = db.Bun().NewSelect().
		Table("templates").
		Where("name = ?", req.TemplateName).
		Scan(ctx, &tpl)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, api.NotFoundErrs("template", req.TemplateName, true)
	case err != nil:
		return nil, fmt.Errorf("fetching template %s from database: %w", req.TemplateName, err)
	}

	permErr, err := AuthZProvider.Get().CanViewTemplate(ctx, user, model.AccessScopeID(tpl.WorkspaceId))
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to check for permissions: %w", err)
	case permErr != nil:
		return nil, authz.SubIfUnauthorized(permErr, api.NotFoundErrs("template", req.TemplateName, true))
	}
	return &apiv1.GetTemplateResponse{Template: &tpl}, nil
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
		req := &apiv1.PatchTemplateConfigRequest{
			TemplateName: req.Template.Name,
			Config:       req.Template.Config,
		}
		_, err = a.PatchTemplateConfig(ctx, req)
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

	// json.Marshal + AsMap is 2x faster than protojson.Marshal or just json.Marshal because
	// marshaling structpb.Struct is really slow.
	configBytes, err := json.Marshal(req.Template.Config.AsMap())
	if err != nil {
		return nil, err
	}

	var inserted templatev1.Template
	err = db.Bun().NewInsert().
		Model(&model.Template{Name: req.Template.Name, WorkspaceID: workspaceID}).
		Value("config", "?", string(configBytes)).
		Returning("*").
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

	configBytes, err := json.Marshal(req.Config.AsMap())
	if err != nil {
		return nil, err
	}

	var updated templatev1.Template
	err = db.Bun().NewUpdate().Model(&model.Template{}).
		Set("config = ?", string(configBytes)).
		Where("name = ?", req.TemplateName).
		Returning("*").
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
