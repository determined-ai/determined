package internal

import (
	"context"
	"fmt"
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

func (a *apiServer) modelByName(name string) (*modelv1.Model, error) {
	m := &modelv1.Model{}
	switch err := a.m.db.QueryProto("get_model", m, name); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "model %s not found", name)
	default:
		return m,
			errors.Wrapf(err, "error fetching model %s from database", name)
	}
}

func (a *apiServer) GetModel(
	_ context.Context, req *apiv1.GetModelRequest) (*apiv1.GetModelResponse, error) {
	m, err := a.modelByName(req.ModelName)
	return &apiv1.GetModelResponse{Model: m},
		errors.Wrapf(err, "error fetching model %s from database", req.ModelName)
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
	_ context.Context, req *apiv1.PatchModelRequest) (*apiv1.PatchModelResponse, error) {
	m, err := a.modelByName(req.Model.Name)
	if err != nil {
		return nil, err
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

func (a *apiServer) GetModelVersion(
	_ context.Context, req *apiv1.GetModelVersionRequest) (*apiv1.GetModelVersionResponse, error) {
	resp := &apiv1.GetModelVersionResponse{}

	switch err := a.m.db.QueryProto("get_model_version", resp, req.ModelName, req.ModelVersion); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "model %s version %d not found", req.ModelName, req.ModelVersion)
	case err != nil:
		fmt.Printf("err = %+v\n", err)
		return nil, err
	}

	return resp, nil
}

func (a *apiServer) PostModelVersion(
	_ context.Context, req *apiv1.PostModelVersionRequest) (*apiv1.PostModelVersionResponse, error) {
	// make sure that the model exists before adding a version
	m, getModelErr := a.modelByName(req.ModelName)
	if getModelErr != nil {
		return nil, getModelErr
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

	if c.State != checkpointv1.Checkpoint_STATE_COMPLETED {
		return nil, errors.Errorf(
			"checkpoint %s is in %s state. checkpoints for model versions must be in a COMPLETED state",
			c.Uuid, c.State,
		)
	}

	respModelVersion := &apiv1.PostModelVersionResponse{}

	err := a.m.db.QueryProto(
		"insert_model_version",
		respModelVersion,
		req.ModelName,
		req.CheckpointUuid,
		time.Now(),
		time.Now(),
	)

	respModelVersion.Model = m
	respModelVersion.Checkpoint = c

	return respModelVersion, errors.Wrapf(err, "error adding model version to model %s", req.ModelName)
}
