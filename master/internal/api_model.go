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
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
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
		errors.Wrapf(err, "error creating model %s in database", req.Model.Name)
}

func (a *apiServer) PatchModel(
	ctx context.Context, req *apiv1.PatchModelRequest) (*apiv1.PatchModelResponse, error) {
	getResp, err := a.GetModel(ctx, &apiv1.GetModelRequest{ModelName: req.Model.Name})
	if err != nil {
		return nil, err
	}

	paths := req.UpdateMask.GetPaths()
	for _, path := range paths {
		switch {
		case path == "model.description":
			getResp.Model.Description = req.Model.Description
		case strings.HasPrefix(path, "model.metadata"):
			getResp.Model.Metadata = req.Model.Metadata
		case !strings.HasPrefix(path, "update_mask"):
			return nil, status.Errorf(
				codes.InvalidArgument,
				"only description and metadata fields are mutable. cannot update %s", path)
		}
	}

	b, err := protojson.Marshal(getResp.Model.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling model.Metadata")
	}

	respModel := &modelv1.Model{}
	err = a.m.db.QueryProto(
		"update_model", respModel, req.Model.Name, getResp.Model.Description, b, time.Now())

	return &apiv1.PatchModelResponse{Model: respModel},
		errors.Wrapf(err, "error updating model %s in database", req.Model.Name)
}

func (a *apiServer) GetModelVersion(
	_ context.Context, req *apiv1.GetModelVersionRequest) (*apiv1.GetModelVersionResponse, error) {
	resp := &apiv1.GetModelVersionResponse{}
	resp.ModelVersion = &modelv1.ModelVersion{}

	switch err := a.m.db.QueryProto(
		"get_model_version", resp.ModelVersion, req.ModelName, req.ModelVersion); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "model %s version %d not found", req.ModelName, req.ModelVersion)
	default:
		return resp, err
	}
}

func (a *apiServer) GetModelVersions(
	ctx context.Context, req *apiv1.GetModelVersionsRequest) (*apiv1.GetModelVersionsResponse, error) {
	getResp, err := a.GetModel(ctx, &apiv1.GetModelRequest{ModelName: req.ModelName})
	if err != nil {
		return nil, err
	}

	resp := &apiv1.GetModelVersionsResponse{Model: getResp.Model}
	if err := a.m.db.QueryProto("get_model_versions", &resp.ModelVersions, req.ModelName); err != nil {
		return nil, err
	}

	a.sort(resp.ModelVersions, req.OrderBy, req.SortBy, apiv1.GetModelVersionsRequest_SORT_BY_VERSION)
	return resp, a.paginate(&resp.Pagination, &resp.ModelVersions, req.Offset, req.Limit)
}

func (a *apiServer) PostModelVersion(
	ctx context.Context, req *apiv1.PostModelVersionRequest) (*apiv1.PostModelVersionResponse, error) {
	// make sure that the model exists before adding a version
	getResp, err := a.GetModel(ctx, &apiv1.GetModelRequest{ModelName: req.ModelName})
	if err != nil {
		return nil, err
	}

	// make sure the checkpoint exists
	c := &checkpointv1.Checkpoint{}

	switch getCheckpointErr := a.m.db.QueryProto("get_checkpoint", c, req.CheckpointUuid); {
	case getCheckpointErr == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "checkpoint %s not found", req.CheckpointUuid)
	case getCheckpointErr != nil:
		return nil, getCheckpointErr
	}

	if c.State != checkpointv1.State_STATE_COMPLETED {
		return nil, errors.Errorf(
			"checkpoint %s is in %s state. checkpoints for model versions must be in a COMPLETED state",
			c.Uuid, c.State,
		)
	}

	respModelVersion := &apiv1.PostModelVersionResponse{}
	respModelVersion.ModelVersion = &modelv1.ModelVersion{}

	err = a.m.db.QueryProto(
		"insert_model_version",
		respModelVersion.ModelVersion,
		req.ModelName,
		req.CheckpointUuid,
	)

	respModelVersion.ModelVersion.Model = getResp.Model
	respModelVersion.ModelVersion.Checkpoint = c

	return respModelVersion, errors.Wrapf(err, "error adding model version to model %s", req.ModelName)
}
