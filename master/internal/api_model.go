package internal

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

func (a *apiServer) GetModel(
	_ context.Context, req *apiv1.GetModelRequest) (*apiv1.GetModelResponse, error) {
	m := &modelv1.Model{}
	switch err := a.m.db.QueryProto("get_model", m, req.ModelName); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "model %s not found", req.ModelName)
	default:
		return &apiv1.GetModelResponse{Model: m},
			errors.Wrapf(err, "error fetching model %s from database", req.ModelName)
	}
}

func (a *apiServer) GetModels(
	_ context.Context, req *apiv1.GetModelsRequest) (*apiv1.GetModelsResponse, error) {
	resp := &apiv1.GetModelsResponse{}
	if err := a.m.db.QueryProto("get_models", &resp.Models); err != nil {
		return nil, err
	}

	a.filter(&resp.Models, func(i int) bool {
		v := resp.Models[i]

		if !strings.Contains(strings.ToLower(v.Name), strings.ToLower(req.Name)) {
			return false
		}

		return strings.Contains(strings.ToLower(v.Description), strings.ToLower(req.Description))
	})

	a.sort(resp.Models, req.OrderBy, req.SortBy, apiv1.GetModelsRequest_SORT_BY_LAST_UPDATED_TIME)
	return resp, a.paginate(&resp.Pagination, &resp.Models, req.Offset, req.Limit)
}

func (a *apiServer) PostModel(
	_ context.Context, req *apiv1.PostModelRequest) (*apiv1.PostModelResponse, error) {
	b, err := protojson.Marshal(req.Model.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling model.Metadata")
	}

	m := &modelv1.Model{}
	err = a.m.db.QueryProto(
		"insert_model", m, req.Model.Name, req.Model.Description, b, time.Now(), time.Now(),
	)

	return &apiv1.PostModelResponse{Model: m},
		errors.Wrapf(err, "error fetching model %s from database", req.Model.Name)
}

func (a *apiServer) PatchModel(
	_ context.Context, req *apiv1.PatchModelRequest) (*apiv1.PatchModelResponse, error) {
	m := &modelv1.Model{}

	switch err := a.m.db.QueryProto("get_model", m, req.Model.Name); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "model %s not found", req.Model.Name)
	case err != nil:
		return nil, status.Errorf(
			codes.Internal, "could not query model %s", req.Model.Name)
	}

	paths := req.UpdateMask.GetPaths()
	for _, path := range paths {
		switch {
		case path == "model.description":
			m.Description = req.Model.Description
		case strings.HasPrefix(path, "model.metadata"):
			m.Metadata = req.Model.Metadata
		case !strings.HasPrefix(path, "update_mask"):
			return nil, status.Errorf(
				codes.InvalidArgument,
				"only description and metadata fields are mutable. cannot update %s", path)
		}
	}

	b, err := protojson.Marshal(m.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling model.Metadata")
	}

	respModel := &modelv1.Model{}
	err = a.m.db.QueryProto(
		"update_model", respModel, req.Model.Name, m.Description, b, time.Now())

	return &apiv1.PatchModelResponse{Model: respModel},
		errors.Wrapf(err, "error updating model %s in database", req.Model.Name)
}
