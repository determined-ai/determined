package internal

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

func (a *apiServer) GetCheckpoint(
	_ context.Context, req *apiv1.GetCheckpointRequest) (*apiv1.GetCheckpointResponse, error) {
	resp := &apiv1.GetCheckpointResponse{}
	resp.Checkpoint = &experimentv1.Checkpoint{}
	switch err := a.m.db.QueryProto("get_checkpoint", resp.Checkpoint, req.CheckpointUuid); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "checkpoint %s not found", req.CheckpointUuid)
	default:
		return resp,
			errors.Wrapf(err, "error fetching checkpoint %s from database", req.CheckpointUuid)
	}
}
