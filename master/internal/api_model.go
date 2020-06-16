package internal

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetModel(
	_ context.Context, req *apiv1.GetModelRequest) (*apiv1.GetModelResponse, error) {
	switch m, err := a.m.db.ModelByName(req.ModelName); err {
	case nil:
		protoTemp, pErr := model.ModelToProto(m)
		return &apiv1.GetModelResponse{Model: protoTemp}, pErr
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "error fetching template from database: %s", req.ModelName)
	default:
		return nil, errors.Wrapf(err, "error fetching template from database: %s", req.ModelName)
	}
}
