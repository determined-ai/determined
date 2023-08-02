package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/template"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/templatev1"
)

func (a *apiServer) GetTemplates(
	ctx context.Context, req *apiv1.GetTemplatesRequest,
) (*apiv1.GetTemplatesResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil,
			status.Errorf(codes.Unauthenticated, "failed to get the user: %s", err)
	}
	resp := &apiv1.GetTemplatesResponse{}
	scopes, err := template.AuthZProvider.Get().ViewableScopes(ctx, user, 0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check for permissions: %s", err)
	}
	if err := a.m.db.QueryProto("get_templates", &resp.Templates); err != nil {
		return nil, errors.Wrap(err, "error fetching templates from database")
	}
	a.filter(&resp.Templates, func(i int) bool {
		if !scopes[model.AccessScopeID(resp.Templates[i].WorkspaceId)] {
			return false
		}
		return strings.Contains(strings.ToLower(resp.Templates[i].Name), strings.ToLower(req.Name))
	})
	a.sort(resp.Templates, req.OrderBy, req.SortBy, apiv1.GetTemplatesRequest_SORT_BY_NAME)
	return resp, a.paginate(&resp.Pagination, &resp.Templates, req.Offset, req.Limit)
}

func (a *apiServer) GetTemplate(
	ctx context.Context, req *apiv1.GetTemplateRequest,
) (*apiv1.GetTemplateResponse, error) {
	t := &templatev1.Template{}
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil,
			status.Errorf(codes.Unauthenticated, "failed to get the user: %s", err)
	}
	switch err := a.m.db.QueryProto("get_template", t, req.TemplateName); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "error fetching template from database: %s", req.TemplateName)
	case nil:
		permErr, err := template.AuthZProvider.Get().CanViewTemplate(
			ctx, user, model.AccessScopeID(t.WorkspaceId))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check for permissions: %s", err)
		}
		if permErr != nil {
			return nil, authz.SubIfUnauthorized(permErr,
				api.NotFoundErrs("template", req.TemplateName, true))
		}
		return &apiv1.GetTemplateResponse{Template: t}, nil
	default:
		return nil,
			errors.Wrapf(err, "error fetching template from database: %s", req.TemplateName)
	}
}

func (a *apiServer) PutTemplate(
	ctx context.Context, req *apiv1.PutTemplateRequest,
) (*apiv1.PutTemplateResponse, error) {
	if req.Template.WorkspaceId != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "setting workspace_id is not supported.")
	}
	var err error
	if _, err = a.m.db.TemplateByName(req.Template.Name); err != nil {
		_, err = a.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: req.Template})
	} else {
		_, err = a.PatchTemplateConfig(ctx,
			&apiv1.PatchTemplateConfigRequest{Config: req.Template.Config})
	}
	return &apiv1.PutTemplateResponse{Template: req.Template},
		errors.Wrapf(err, "error putting template")
}

func (a *apiServer) PostTemplate(
	ctx context.Context, req *apiv1.PostTemplateRequest,
) (*apiv1.PostTemplateResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil,
			status.Errorf(codes.Unauthenticated, "failed to get the user: %s", err)
	}
	config, err := protojson.Marshal(req.Template.Config)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid config provided: %s", err.Error())
	}
	workspaceID := model.DefaultWorkspaceID
	if req.Template.WorkspaceId != 0 {
		workspaceID = int(req.Template.WorkspaceId)
	}
	notFoundErr := api.NotFoundErrs("workspace", fmt.Sprint(workspaceID), true)
	var exists bool
	err = db.Bun().NewSelect().ColumnExpr("1").Table("workspaces").
		Where("id = ?", workspaceID).
		Where("archived = false").
		Limit(1).
		Scan(ctx, &exists)
	if err != nil {
		return nil, errors.Wrapf(err, "error checking workspace %d", workspaceID)
	}
	if !exists {
		return nil, notFoundErr
	}

	permErr, err := template.AuthZProvider.Get().CanCreateTemplate(ctx,
		user, model.AccessScopeID(workspaceID))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check for permissions: %s", err)
	}
	if permErr != nil {
		return nil, permErr
	}

	res := apiv1.PostTemplateResponse{Template: &templatev1.Template{}}
	switch err := a.m.db.QueryProto(
		"insert_template", res.Template, req.Template.Name, config, workspaceID,
	); err {
	case nil:
		return &res, nil
	default:
		return nil, status.Errorf(codes.Internal, "error posting template %s to db: %s",
			req.Template.Name, err.Error())
	}
}

func (a *apiServer) PatchTemplateConfig(
	ctx context.Context, req *apiv1.PatchTemplateConfigRequest,
) (*apiv1.PatchTemplateConfigResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil,
			status.Errorf(codes.Unauthenticated, "failed to get the user: %s", err)
	}
	config, err := protojson.Marshal(req.Config)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid config provided: %s", err.Error())
	}
	// fails if we don't have read access.
	temp, err := a.GetTemplate(ctx, &apiv1.GetTemplateRequest{TemplateName: req.TemplateName})
	if err != nil {
		return nil, err
	}
	permErr, err := template.AuthZProvider.Get().CanUpdateTemplate(
		ctx, user, model.AccessScopeID(temp.Template.WorkspaceId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check for permissions: %s", err)
	}
	if permErr != nil {
		return nil, permErr
	}

	template := templatev1.Template{}
	switch err := a.m.db.QueryProto(
		"update_template", &template, req.TemplateName, config,
	); err {
	case nil:
		return &apiv1.PatchTemplateConfigResponse{Template: &template}, nil
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "error updating template %s to db: %s", req.TemplateName, err.Error())
	default:
		return nil, status.Errorf(codes.Internal, "failed to update template: %s", err.Error())
	}
}

func (a *apiServer) DeleteTemplate(
	ctx context.Context, req *apiv1.DeleteTemplateRequest,
) (*apiv1.DeleteTemplateResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil,
			status.Errorf(codes.Unauthenticated, "failed to get the user: %s", err)
	}
	temp, err := a.GetTemplate(ctx, &apiv1.GetTemplateRequest{TemplateName: req.TemplateName})
	if err != nil {
		return nil, err
	}
	permErr, err := template.AuthZProvider.Get().CanDeleteTemplate(
		ctx, user, model.AccessScopeID(temp.Template.WorkspaceId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check for permissions: %s", err)
	}
	if permErr != nil {
		return nil, permErr
	}
	switch err := a.m.db.DeleteTemplate(req.TemplateName); err {
	case nil:
		return &apiv1.DeleteTemplateResponse{}, nil
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "error fetching template from database: %s", req.TemplateName)
	default:
		return nil, err
	}
}
