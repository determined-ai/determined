package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

func errCheckpointNotFound(id string) error {
	return status.Errorf(codes.NotFound, "checkpoint not found: %s", id)
}

func errCheckpointsNotFound(ids []string) error {
	tmp := make([]string, len(ids))
	for i, id := range ids {
		tmp[i] = id
	}
	sort.Strings(tmp)
	return status.Errorf(codes.NotFound, "checkpoints not found: %s", strings.Join(tmp, ", "))
}

func (a *apiServer) GetCheckpoint(
	ctx context.Context, req *apiv1.GetCheckpointRequest,
) (*apiv1.GetCheckpointResponse, error) {
	resp := &apiv1.GetCheckpointResponse{}
	resp.Checkpoint = &checkpointv1.Checkpoint{}

	if err := a.m.db.QueryProto(
		"get_checkpoint", resp.Checkpoint, req.CheckpointUuid); errors.Is(err, db.ErrNotFound) {
		return nil, errCheckpointNotFound(req.CheckpointUuid)
	} else if err != nil {
		return nil,
			errors.Wrapf(err, "error fetching checkpoint %s from database", req.CheckpointUuid)
	}

	taskID := model.TaskID(resp.Checkpoint.TaskId)
	if err := a.canDoActionsOnTask(
		ctx, taskID, expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		if err == errTaskNotFound(taskID) {
			err = errCheckpointNotFound(req.CheckpointUuid)
		}
		return nil, err
	}
	return resp, nil
}

// TODO...
func (a *apiServer) DeleteCheckpoints(
	ctx context.Context,
	req *apiv1.DeleteCheckpointsRequest,
) (*apiv1.DeleteCheckpointsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	conv := &protoconverter.ProtoConverter{}
	checkpointsToDelete := conv.ToUUIDList(req.CheckpointUuids)
	if cErr := conv.Error(); cErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, "converting checkpoint: %s", cErr)
	}

	registeredCheckpointUUIDs, err := a.m.db.GetRegisteredCheckpoints(checkpointsToDelete)
	if err != nil {
		return nil, err
	}

	if len(registeredCheckpointUUIDs) > 0 {
		return nil, status.Errorf(codes.InvalidArgument,
			"this subset of checkpoints provided are in the model registry and cannot be deleted: %v.",
			registeredCheckpointUUIDs)
	}

	addr := actor.Addr(fmt.Sprintf("checkpoints-gc-%s", uuid.New().String()))

	taskSpec := *a.m.taskSpec

	jobID := model.NewJobID()
	if err = a.m.db.AddJob(&model.Job{
		JobID:   jobID,
		JobType: model.JobTypeCheckpointGC,
		OwnerID: &curUser.ID,
	}); err != nil {
		return nil, fmt.Errorf("persisting new job: %w", err)
	}

	groupCUUIDsByEIDs, err := a.m.db.GroupCheckpointUUIDsByExperimentID(checkpointsToDelete)
	if err != nil {
		return nil, err
	}

	// Are we missing any checkpoints?
	checkpointsRequested := make(map[string]bool)
	for _, c := range req.CheckpointUuids {
		checkpointsRequested[c] = true
	}
	for _, expIDcUUIDs := range groupCUUIDsByEIDs {
		for _, c := range strings.Split(expIDcUUIDs.CheckpointUUIDSStr, ",") {
			checkpointsRequested[c] = false
		}
	}
	var notFoundCheckpoints []string
	for c, notFound := range checkpointsRequested {
		if notFound {
			notFoundCheckpoints = append(notFoundCheckpoints, c)
		}
	}

	for _, expIDcUUIDs := range groupCUUIDsByEIDs {
		exp, err := a.m.db.ExperimentByID(expIDcUUIDs.ExperimentID)
		if err != nil {
			return nil, err
		}
		if ok, err := expauth.AuthZProvider.Get().CanGetExperiment(*curUser, exp); err != nil {
			return nil, err
		} else if !ok {
			notFoundCheckpoints = append(notFoundCheckpoints,
				strings.Split(expIDcUUIDs.CheckpointUUIDSStr, ",")...)
			continue
		}
		if err := expauth.AuthZProvider.Get().CanEditExperiment(*curUser, exp); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		// Don't delete any checkpoints when we already haven't found a checkpoint.
		// We will however keep looking to gather all checkpoints a user can't find.
		if len(notFoundCheckpoints) > 0 {
			continue
		}

		agentUserGroup, err := user.GetAgentUserGroup(curUser.ID, exp)
		if err != nil {
			return nil, err
		}

		jobSubmissionTime := time.Now().UTC().Truncate(time.Millisecond)
		taskID := model.NewTaskID()
		conv := &protoconverter.ProtoConverter{}
		checkpointUUIDs := conv.ToUUIDList(strings.Split(expIDcUUIDs.CheckpointUUIDSStr, ","))
		ckptGCTask := newCheckpointGCTask(
			a.m.rm, a.m.db, a.m.taskLogger, taskID, jobID, jobSubmissionTime, taskSpec, exp.ID,
			exp.Config.AsLegacy(), checkpointUUIDs, false, agentUserGroup, curUser, nil,
		)
		a.m.system.MustActorOf(addr, ckptGCTask)
	}

	if len(notFoundCheckpoints) > 0 {
		return nil, errCheckpointsNotFound(notFoundCheckpoints)
	}
	return &apiv1.DeleteCheckpointsResponse{}, nil
}

func (a *apiServer) PostCheckpointMetadata(
	ctx context.Context, req *apiv1.PostCheckpointMetadataRequest,
) (*apiv1.PostCheckpointMetadataResponse, error) {
	currCheckpoint := &checkpointv1.Checkpoint{}
	if err := a.m.db.QueryProto(
		"get_checkpoint", currCheckpoint, req.Checkpoint.Uuid); errors.Is(err, db.ErrNotFound) {
		return nil, errCheckpointNotFound(req.Checkpoint.Uuid)
	} else if err != nil {
		return nil,
			errors.Wrapf(err, "error fetching checkpoint %s from database", req.Checkpoint.Uuid)
	}

	taskID := model.TaskID(currCheckpoint.TaskId)
	if err := a.canDoActionsOnTask(
		ctx, taskID, expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		if err == errTaskNotFound(taskID) {
			err = errCheckpointNotFound(req.Checkpoint.Uuid)
		}
		return nil, err
	}

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
