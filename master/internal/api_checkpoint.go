package internal

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

func (a *apiServer) GetCheckpoint(
	_ context.Context, req *apiv1.GetCheckpointRequest) (*apiv1.GetCheckpointResponse, error) {
	resp := &apiv1.GetCheckpointResponse{}
	resp.Checkpoint = &checkpointv1.Checkpoint{}
	switch err := a.m.db.QueryProto("get_checkpoint", resp.Checkpoint, req.CheckpointUuid); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "checkpoint %s not found", req.CheckpointUuid)
	case err != nil:
		return nil, errors.Wrapf(err, "error fetching checkpoint %s from database", req.CheckpointUuid)
	default:
		return resp, nil
	}
}

func (a *apiServer) CreateCheckpoint(
	_ context.Context, req *apiv1.CreateCheckpointRequest,
) (*apiv1.CreateCheckpointResponse, error) {
	log.Infof("adding checkpoint %s (trial %d, batch %d)",
		req.Checkpoint.Uuid, req.Checkpoint.TrialId, req.Checkpoint.BatchNumber)
	modelC, err := model.CheckpointFromProto(req.Checkpoint)
	if err != nil {
		return nil, errors.Wrapf(
			err, "error adding checkpoint %s (trial %d, batch %d) in database",
			req.Checkpoint.Uuid, req.Checkpoint.TrialId, req.Checkpoint.BatchNumber)
	}
	err = a.m.db.AddCheckpoint(modelC)
	if err != nil {
		return nil, errors.Wrapf(err, "error adding checkpoint %s (trial %d, batch %d) in database",
			req.Checkpoint.Uuid, req.Checkpoint.TrialId, req.Checkpoint.BatchNumber)
	}
	return &apiv1.CreateCheckpointResponse{Checkpoint: req.Checkpoint}, nil
}

func (a *apiServer) PatchCheckpoint(
	_ context.Context, req *apiv1.PatchCheckpointRequest,
) (*apiv1.PatchCheckpointResponse, error) {
	log.Infof("patching checkpoint %s (trial %d, batch %d) state %s",
		req.Checkpoint.Uuid, req.Checkpoint.TrialId, req.Checkpoint.BatchNumber,
		req.Checkpoint.State)
	modelC, err := model.CheckpointFromProto(req.Checkpoint)
	if err != nil {
		return nil, errors.Wrapf(
			err, "error patching checkpoint %s (trial %d, batch %d) in database",
			req.Checkpoint.Uuid, req.Checkpoint.TrialId, req.Checkpoint.BatchNumber)
	}
	err = a.m.db.UpdateCheckpoint(
		int(req.Checkpoint.TrialId), int(req.Checkpoint.BatchNumber), *modelC)
	if err != nil {
		return nil, errors.Wrapf(err, "error patching checkpoint %s (trial %d, batch %d) in database",
			req.Checkpoint.Uuid, req.Checkpoint.TrialId, req.Checkpoint.BatchNumber)
	}
	return &apiv1.PatchCheckpointResponse{Checkpoint: req.Checkpoint}, nil
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
	if err != nil {
		return nil, errors.Wrapf(err, "error updating checkpoint %s in database", req.Checkpoint.Uuid)
	}

	return &apiv1.PostCheckpointMetadataResponse{Checkpoint: currCheckpoint}, nil
}
