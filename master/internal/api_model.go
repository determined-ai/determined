package internal

import (
	"context"
	"fmt"
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

	existingModel := &modelv1.Model{}
	switch err := a.m.db.QueryProto("get_model", existingModel, req.Model.Name); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "model %s not found", req.Model.Name)
	default:
		fmt.Printf("req = %+v\n", req)

		desiredModel := &modelv1.Model{}
		paths := req.UpdateMask.GetPaths()
		for _, v := range paths {
			fmt.Printf("v = %+v\n", v)
			if v == "model.description" {
				desiredModel.Description = req.Model.Description
			} else {
				desiredModel.Description = existingModel.Description
			}
			if v == "model.metadata" {
				fmt.Println("UPDATE Meta")
				desiredModel.Metadata = req.Model.Metadata
			} else {
				desiredModel.Metadata = existingModel.Metadata
			}
		}

		respModel := &modelv1.Model{}
		b, err := protojson.Marshal(desiredModel.Metadata)
		if err != nil {
			return nil, errors.Wrap(err, "error marshaling model.Metadata")
		}

		err = a.m.db.QueryProto("update_model", respModel, req.Model.Name, desiredModel.Description, b, time.Now())
		return &apiv1.PatchModelResponse{Model: respModel},
			errors.Wrapf(err, "error fetching model %s from database", req.Model.Name)
	}
}
