package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
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
	ctx context.Context,
	req *apiv1.DeleteCheckpointsRequest) (*apiv1.DeleteCheckpointsResponse, error) {
	checkpointsToDelete := req.CheckpointUuids
	registeredCheckpointUUIDs, err := a.m.db.FilterForRegisteredCheckpoints(checkpointsToDelete)
	if err != nil {
		return nil, status.Errorf(
			codes.Unknown, "error getting list of checkpoints in model registry")
	}

	// return 400 if model registry checkpoints and include all the model registry checkpoints
	if len(registeredCheckpointUUIDs) > 0 {
		return nil, status.Errorf(codes.InvalidArgument,
			"This subset of list of checkpoints provided are in the model registry: %v cannot be deleted.",
			registeredCheckpointUUIDs)
	}

	taskSpec := *a.m.taskSpec

	addr := actor.Addr(fmt.Sprintf("checkpoints-gc-%s", uuid.New().String()))

	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, status.Errorf(
			codes.Unknown, "cannot get current user for the current session")
	}

	agentUserGroup, err := a.m.db.AgentUserGroup(curUser.ID)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"cannot find user and group information for user %s: %s",
			curUser.Username,
			err,
		)
	}
	if agentUserGroup == nil {
		agentUserGroup = &a.m.config.Security.DefaultTask
	}

	taskSpec.AgentUserGroup = agentUserGroup

	jobID := model.NewJobID()
	if err = a.m.db.AddJob(&model.Job{
		JobID:   jobID,
		JobType: model.JobTypeCheckpointGC,
		OwnerID: &curUser.ID,
	}); err != nil {
		return nil, status.Error(codes.Internal, "persisting new job")
	}

	groupeIDcUUIDS, err := a.m.db.GetExpIDsUsingCheckpointUUIDs(checkpointsToDelete)

	if err != nil {
		print(err)
		return nil, err
	}

	for _, expIDcUUIDs := range groupeIDcUUIDS {
		exp, err1 := a.m.db.ExperimentByID(expIDcUUIDs.EID)

		if err1 != nil {
			return nil, err
		}

		CUUIDS := strings.Split(expIDcUUIDs.CUUIDsString, ",")
		jsonVCheckpoints, _ := json.Marshal(CUUIDS)

		if gcErr := a.m.system.MustActorOf(addr, &checkpointGCTask{
			taskID:            model.NewTaskID(),
			jobID:             jobID,
			jobSubmissionTime: time.Now().UTC().Truncate(time.Millisecond),
			GCCkptSpec: tasks.GCCkptSpec{
				Base:         taskSpec,
				ExperimentID: exp.ID,
				LegacyConfig: exp.Config.AsLegacy(),
				ToDelete:     jsonVCheckpoints,
			},
			rm: a.m.rm,
			db: a.m.db,

			taskLogger: a.m.taskLogger,
		}).AwaitTermination(); gcErr != nil {
			return nil, status.Error(codes.Aborted, "failed to GC checkpoints given by user")
		}
	}

	err = a.m.db.MarkCheckpointsDeleted(checkpointsToDelete)
	if err != nil {
		return nil, status.Error(codes.Aborted, "marking GC checkpoints as deleted")
	}

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
