package internal

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/templatev1"
)

func toProtoTemplate(template model.Template) (*templatev1.Template, error) {
	configStruct := &structpb.Struct{}
	err := protojson.Unmarshal(template.Config, configStruct)
	return &templatev1.Template{Name: template.Name, Config: configStruct},
		errors.Wrapf(err, "error parsing template: %s", template.Name)
}

func toModelProto(template *templatev1.Template) (model.Template, error) {
	bytes, err := protojson.Marshal(template.Config)
	return model.Template{Name: template.Name, Config: bytes},
		errors.Wrapf(err, "error parsing template: %s", template.Name)
}

func (a *apiServer) GetTemplates(
	_ context.Context, req *apiv1.GetTemplatesRequest) (*apiv1.GetTemplatesResponse, error) {
	resp := &apiv1.GetTemplatesResponse{}
	templates, err := a.m.db.TemplateList()
	if err != nil {
		return nil, errors.Wrap(err, "error fetching templates from database")
	}
	for _, template := range templates {
		protoTemp, err := toProtoTemplate(template)
		if err != nil {
			return nil, err
		}
		resp.Templates = append(resp.Templates, protoTemp)
	}
	sort.Slice(resp.Templates, func(i, j int) bool {
		t1, t2 := resp.Templates[i], resp.Templates[j]
		if req.OrderBy == apiv1.GetTemplatesRequest_ORDER_BY_DESC {
			t1, t2 = t2, t1
		}
		return t1.Name < t2.Name
	})
	return resp, nil
}

func (a *apiServer) GetTemplate(
	_ context.Context, req *apiv1.GetTemplateRequest) (*apiv1.GetTemplateResponse, error) {
	switch template, err := a.m.db.TemplateByName(req.TemplateName); err {
	case nil:
		protoTemp, pErr := toProtoTemplate(template)
		return &apiv1.GetTemplateResponse{Template: protoTemp}, pErr
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "error fetching template from database: %s", req.TemplateName)
	default:
		return nil, errors.Wrapf(err, "error fetching template from database: %s", req.TemplateName)
	}
}

func (a *apiServer) PutTemplate(
	_ context.Context, req *apiv1.PutTemplateRequest) (*apiv1.PutTemplateResponse, error) {
	template, err := toModelProto(req.Template)
	if err != nil {
		return nil, err
	}
	err = a.m.db.UpsertTemplate(&template)
	return &apiv1.PutTemplateResponse{Template: req.Template}, err
}

func (a *apiServer) DeleteTemplate(
	_ context.Context, req *apiv1.DeleteTemplateRequest) (*apiv1.DeleteTemplateResponse, error) {
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
