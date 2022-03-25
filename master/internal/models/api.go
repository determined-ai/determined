package models

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/ckpts"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

type ModelsAPI struct{}

// // XXX: replace api_models.go::apiServer.GetModel() with this.
// func (s *ModelsAPI) GetModel(
// 	ctx context.Context, req *apiv1.GetModelRequest) (*apiv1.GetModelResponse, error) {
// 	m, err := ByName(ctx, req.ModelName)
// 	if err != nil {
// 		return nil, err
// 	}
// 	pc := protoutils.ProtoConverter{}
// 	pmv := m.ToProto(&pc)
// 	return &apiv1.GetModelResponse{Model: &pmv}, pc.Error()
// }

func (s *ModelsAPI) GetModelVersion(
	ctx context.Context, req *apiv1.GetModelVersionRequest,
) (*apiv1.GetModelVersionResponse, error) {
	m, err := ByName(ctx, req.ModelName)
	switch {
	case err == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "model %s not found", req.ModelName)
	case err != nil:
		return nil, err
	}

	// Note: we are passing req.ModelVersion as an id intentionally; it's named wrong.
	// XXX: make Version accept int or int32, with generics
	mv, err := VersionByID(ctx, int(req.ModelVersion))
	switch {
	case err == db.ErrNotFound:
		return nil, status.Errorf(
			// XXX: wait I thought req.ModelVersion was secretly the model.id?
			// XXX: That would also imply that the protobuf request fields are overdefined.
			codes.NotFound, "model %s version %d not found", req.ModelName, req.ModelVersion)
	case err != nil:
		return nil, err
	}

	ckpt, err := ckpts.ByIDExpanded(ctx, mv.CheckpointID)
	switch {
	case err == db.ErrNotFound:
		// This should not happen.
		return nil, status.Errorf(codes.Internal, "checkpoint missing")
	case err != nil:
		return nil, err
	}

	resp := &apiv1.GetModelVersionResponse{}
	pc := protoutils.ProtoConverter{}
	mvv1 := mv.ToProto(&pc, m, ckpt)
	resp.ModelVersion = &mvv1

	return resp, pc.Error()
}
