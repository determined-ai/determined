package internal

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

func (a *apiServer) GetCheckpoint(
	_ context.Context, req *apiv1.GetCheckpointRequest) (*apiv1.GetCheckpointResponse, error) {
	resp := &apiv1.GetCheckpointResponse{}
	resp.Checkpoint = &checkpointv1.Checkpoint{}
	switch err := a.m.db.QueryProto("get_checkpoint", resp.Checkpoint, req.CheckpointUuid); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "checkpoint %s not found", req.CheckpointUuid)
	default:
		return resp,
			errors.Wrapf(err, "error fetching checkpoint %s from database", req.CheckpointUuid)
	}
}

func (a *apiServer) DeleteCheckpoints(
	_ context.Context, req *apiv1.DeleteCheckpointsRequest) (*apiv1.DeleteCheckpointsResponse, error) {
	checkpoints := req.CheckpointUuids
	// figure otu if
	return &apiv1.DeleteCheckpointsResponse{}, nil
}

func (a *apiServer) PostCheckpointMetadata(
	ctx context.Context, req *apiv1.PostCheckpointMetadataRequest,
) (*apiv1.PostCheckpointMetadataResponse, error) {
	getResp, err := a.GetCheckpoint(ctx,
		&apiv1.GetCheckpointRequest{CheckpointUuid: req.Checkpoint.Uuid})
	if err != nil {
		return nil, err
	}

	currCheckpoint := getResp.Checkpoint

	currMeta, err := protojson.Marshal(currCheckpoint.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling database checkpoint metadata")
	}

	newMeta, err := protojson.Marshal(req.Checkpoint.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling request checkpoint metadata")
	}

	currCheckpoint.Metadata = req.Checkpoint.Metadata
	log.Infof("checkpoint (%s) metadata changing from %s to %s",
		req.Checkpoint.Uuid, currMeta, newMeta)
	err = a.m.db.QueryProto("update_checkpoint_metadata",
		&checkpointv1.Checkpoint{}, req.Checkpoint.Uuid, newMeta)

	return &apiv1.PostCheckpointMetadataResponse{Checkpoint: currCheckpoint},
		errors.Wrapf(err, "error updating checkpoint %s in database", req.Checkpoint.Uuid)
}
