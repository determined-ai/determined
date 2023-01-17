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

func (m *Master) canDoActionOnCheckpoint(
	ctx context.Context,
	curUser model.User,
	id string,
	action func(context.Context, model.User, *model.Experiment) error,
) error {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	checkpoint, err := m.db.CheckpointByUUID(uuid)
	if err != nil {
		return err
	} else if checkpoint == nil {
		return errCheckpointNotFound(id)
	}
	if checkpoint.CheckpointTrainingMetadata.ExperimentID == 0 {
		return nil // TODO(nick) add authz for other task types.
	}
	exp, err := m.db.ExperimentByID(checkpoint.CheckpointTrainingMetadata.ExperimentID)
	if err != nil {
		return err
	}

	if ok, err := expauth.AuthZProvider.Get().CanGetExperiment(ctx, curUser, exp); err != nil {
		return err
	} else if !ok {
		return errCheckpointNotFound(id)
	}
	if err := action(ctx, curUser, exp); err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	return nil
}

/* func (m *Master) canDoActionOnCheckpointThroughModel(ctx context.Context, curUser model.User,
	id string) error {

	modelauth.AuthZProvider.Get().CanGetModel
	return nil
}. */
func (a *apiServer) GetCheckpoint(
	ctx context.Context, req *apiv1.GetCheckpointRequest,
) (*apiv1.GetCheckpointResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	errE := a.m.canDoActionOnCheckpoint(ctx, *curUser, req.CheckpointUuid,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts)
	// Nikita TODO: get model from checkpointUuid and then use CanGetModel
	// errM := a.m.canDoActionOnCheckpointThroughModel(ctx, *curUser, req.CheckpointUuid,)

	if errE != nil { // && errM != nil { // allow downloading checkpoint through model of experiment
		return nil, errE
	}

	resp := &apiv1.GetCheckpointResponse{}
	resp.Checkpoint = &checkpointv1.Checkpoint{}

	if err = a.m.db.QueryProto(
		"get_checkpoint", resp.Checkpoint, req.CheckpointUuid); err != nil {
		return resp,
			errors.Wrapf(err, "error fetching checkpoint %s from database", req.CheckpointUuid)
	}

	return resp, nil
}

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

	// Get checkpoints IDs not associated to any experiments.
	checkpointsRequested := make(map[string]bool)
	for _, c := range req.CheckpointUuids {
		checkpointsRequested[c] = false
	}
	for _, expIDcUUIDs := range groupCUUIDsByEIDs {
		for _, c := range strings.Split(expIDcUUIDs.CheckpointUUIDSStr, ",") {
			checkpointsRequested[c] = true
		}
	}
	var notFoundCheckpoints []string
	for c, found := range checkpointsRequested {
		if !found {
			notFoundCheckpoints = append(notFoundCheckpoints, c)
		}
	}

	// Get experiments for all checkpoints and validate
	// that the user has permission to view and edit.
	exps := make([]*model.Experiment, len(groupCUUIDsByEIDs))
	for i, expIDcUUIDs := range groupCUUIDsByEIDs {
		exp, err := a.m.db.ExperimentByID(expIDcUUIDs.ExperimentID)
		if err != nil {
			return nil, err
		}
		var ok bool
		if ok, err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, exp); err != nil {
			return nil, err
		} else if !ok {
			notFoundCheckpoints = append(notFoundCheckpoints,
				strings.Split(expIDcUUIDs.CheckpointUUIDSStr, ",")...)
			continue
		}
		if err = expauth.AuthZProvider.Get().CanEditExperiment(ctx, *curUser, exp); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		exps[i] = exp
	}
	if len(notFoundCheckpoints) > 0 {
		return nil, errCheckpointsNotFound(notFoundCheckpoints)
	}

	// Submit checkpoint GC tasks for all checkpoints.
	for i, expIDcUUIDs := range groupCUUIDsByEIDs {
		agentUserGroup, err := user.GetAgentUserGroup(curUser.ID, exps[i])
		if err != nil {
			return nil, err
		}

		jobSubmissionTime := time.Now().UTC().Truncate(time.Millisecond)
		taskID := model.NewTaskID()
		conv := &protoconverter.ProtoConverter{}
		checkpointUUIDs := conv.ToUUIDList(strings.Split(expIDcUUIDs.CheckpointUUIDSStr, ","))
		ckptGCTask := newCheckpointGCTask(
			a.m.rm, a.m.db, a.m.taskLogger, taskID, jobID, jobSubmissionTime, taskSpec, exps[i].ID,
			exps[i].Config, checkpointUUIDs, false, agentUserGroup, curUser, nil,
		)
		a.m.system.MustActorOf(addr, ckptGCTask)
	}

	return &apiv1.DeleteCheckpointsResponse{}, nil
}

func (a *apiServer) PostCheckpointMetadata(
	ctx context.Context, req *apiv1.PostCheckpointMetadataRequest,
) (*apiv1.PostCheckpointMetadataResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = a.m.canDoActionOnCheckpoint(ctx, *curUser, req.Checkpoint.Uuid,
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	currCheckpoint := &checkpointv1.Checkpoint{}
	if err = a.m.db.QueryProto("get_checkpoint", currCheckpoint, req.Checkpoint.Uuid); err != nil {
		return nil,
			errors.Wrapf(err, "error fetching checkpoint %s from database", req.Checkpoint.Uuid)
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
