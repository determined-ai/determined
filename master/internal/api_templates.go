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
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/templatev1"
)

func (a *apiServer) GetTemplates(
	_ context.Context, req *apiv1.GetTemplatesRequest,
) (*apiv1.GetTemplatesResponse, error) {
	resp := &apiv1.GetTemplatesResponse{}
	if err := a.m.db.QueryProto("get_templates", &resp.Templates); err != nil {
		return nil, errors.Wrap(err, "error fetching templates from database")
	}
	a.filter(&resp.Templates, func(i int) bool {
		return strings.Contains(strings.ToLower(resp.Templates[i].Name), strings.ToLower(req.Name))
	})
	a.sort(resp.Templates, req.OrderBy, req.SortBy, apiv1.GetTemplatesRequest_SORT_BY_NAME)
	return resp, a.paginate(&resp.Pagination, &resp.Templates, req.Offset, req.Limit)
}

func (a *apiServer) GetTemplate(
	_ context.Context, req *apiv1.GetTemplateRequest,
) (*apiv1.GetTemplateResponse, error) {
	t := &templatev1.Template{}
	switch err := a.m.db.QueryProto("get_template", t, req.TemplateName); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "error fetching template from database: %s", req.TemplateName)
	default:
		return &apiv1.GetTemplateResponse{Template: t},
			errors.Wrapf(err, "error fetching template from database: %s", req.TemplateName)
	}
}

func (a *apiServer) PutTemplate(
	_ context.Context, req *apiv1.PutTemplateRequest,
) (*apiv1.PutTemplateResponse, error) {
	config, err := protojson.Marshal(req.Template.Config)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid config provided: %s", err.Error())
	}
	if req.Template.WorkspaceId != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "setting workspace_id is not supported.")
	}
	err = a.m.db.QueryProto("put_template", req.Template, req.Template.Name, config)
	return &apiv1.PutTemplateResponse{Template: req.Template},
		errors.Wrapf(err, "error putting template")
}

func (a *apiServer) PostTemplate(
	ctx context.Context, req *apiv1.PostTemplateRequest,
) (*apiv1.PostTemplateResponse, error) {
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
	config, err := protojson.Marshal(req.Config)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid config provided: %s", err.Error())
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
	_ context.Context, req *apiv1.DeleteTemplateRequest,
) (*apiv1.DeleteTemplateResponse, error) {
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
