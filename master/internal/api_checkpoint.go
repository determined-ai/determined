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
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	modelauth "github.com/determined-ai/determined/master/internal/model"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

func errCheckpointsNotFound(ids []string) error {
	tmp := make([]string, len(ids))
	for i, id := range ids {
		tmp[i] = id
	}
	sort.Strings(tmp)
	return api.NotFoundErrs("checkpoints", strings.Join(tmp, ", "), true)
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
		return api.NotFoundErrs("checkpoint", id, true)
	}
	if checkpoint.CheckpointTrainingMetadata.ExperimentID == 0 {
		return nil // TODO(nick) add authz for other task types.
	}
	exp, err := db.ExperimentByID(ctx, checkpoint.CheckpointTrainingMetadata.ExperimentID)
	if err != nil {
		return err
	}

	if err := expauth.AuthZProvider.Get().CanGetExperiment(ctx, curUser, exp); err != nil {
		return authz.SubIfUnauthorized(err, api.NotFoundErrs("checkpoint", id, true))
	}
	if err := action(ctx, curUser, exp); err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	return nil
}

func (m *Master) canDoActionOnCheckpointThroughModel(
	ctx context.Context, curUser model.User, ckptID string,
) error {
	ckptUUID, err := uuid.Parse(ckptID)
	if err != nil {
		return err
	}

	modelIDs, err := db.GetModelIDsAssociatedWithCheckpoint(ctx, ckptUUID)
	if err != nil {
		return err
	}
	if len(modelIDs) == 0 {
		// if length of model ids is zero then permission denied
		// so return checkpoint not found.
		return api.NotFoundErrs("checkpoint", ckptID, true)
	}

	var errCanGetModel error
	for _, modelID := range modelIDs {
		model := &modelv1.Model{}
		err = m.db.QueryProto("get_model_by_id", model, modelID)
		if err != nil {
			return err
		}
		if errCanGetModel = modelauth.AuthZProvider.Get().CanGetModel(
			ctx, curUser, model, model.WorkspaceId); errCanGetModel == nil {
			return nil
		}
	}
	// we get to this return when there are no models belonging
	// to a workspace where user has permissions.
	return authz.SubIfUnauthorized(errCanGetModel, api.NotFoundErrs("checkpoint", ckptID, true))
}

func (a *apiServer) GetCheckpoint(
	ctx context.Context, req *apiv1.GetCheckpointRequest,
) (*apiv1.GetCheckpointResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	errE := a.m.canDoActionOnCheckpoint(ctx, *curUser, req.CheckpointUuid,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts)

	if errE != nil {
		errM := a.m.canDoActionOnCheckpointThroughModel(ctx, *curUser, req.CheckpointUuid)
		if errM != nil {
			return nil, errE
		}
	}

	resp := &apiv1.GetCheckpointResponse{}
	resp.Checkpoint = &checkpointv1.Checkpoint{}

	if err := a.m.db.QueryProto(
		"get_checkpoint", resp.Checkpoint, req.CheckpointUuid); err != nil {
		return resp,
			errors.Wrapf(err, "error fetching checkpoint %s from database", req.CheckpointUuid)
	}

	return resp, nil
}

func (a *apiServer) checkpointsRBACEditCheck(
	ctx context.Context, uuids []uuid.UUID,
) ([]*model.Experiment, []*db.ExperimentCheckpointGrouping, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, nil, err
	}

	groupCUUIDsByEIDs, err := a.m.db.GroupCheckpointUUIDsByExperimentID(uuids)
	if err != nil {
		return nil, nil, err
	}

	// Get checkpoints IDs not associated to any experiments.
	checkpointsRequested := make(map[string]bool)
	for _, c := range uuids {
		checkpointsRequested[c.String()] = false
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
		exp, err := db.ExperimentByID(ctx, expIDcUUIDs.ExperimentID)
		if err != nil {
			return nil, nil, err
		}
		err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, exp)
		if authz.IsPermissionDenied(err) {
			notFoundCheckpoints = append(notFoundCheckpoints,
				strings.Split(expIDcUUIDs.CheckpointUUIDSStr, ",")...)
			continue
		} else if err != nil {
			return nil, nil, err
		}
		if err = expauth.AuthZProvider.Get().CanEditExperiment(ctx, *curUser, exp); err != nil {
			return nil, nil, status.Error(codes.PermissionDenied, err.Error())
		}

		exps[i] = exp
	}

	if len(notFoundCheckpoints) > 0 {
		return nil, nil, errCheckpointsNotFound(notFoundCheckpoints)
	}

	return exps, groupCUUIDsByEIDs, nil
}

func (a *apiServer) PatchCheckpoints(
	ctx context.Context,
	req *apiv1.PatchCheckpointsRequest,
) (*apiv1.PatchCheckpointsResponse, error) {
	var uuidStrings []string
	for _, c := range req.Checkpoints {
		uuidStrings = append(uuidStrings, c.Uuid)
	}

	conv := &protoconverter.ProtoConverter{}
	uuids := conv.ToUUIDList(uuidStrings)
	if cErr := conv.Error(); cErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, "converting checkpoint: %s", cErr)
	}

	if _, _, err := a.checkpointsRBACEditCheck(ctx, uuids); err != nil {
		return nil, err
	}

	registeredCheckpointUUIDs, err := a.m.db.GetRegisteredCheckpoints(uuids)
	if err != nil {
		return nil, err
	}
	if len(registeredCheckpointUUIDs) > 0 {
		return nil, status.Errorf(codes.InvalidArgument,
			"this subset of checkpoints provided are in the model registry and cannot be deleted: %v.",
			registeredCheckpointUUIDs)
	}

	err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var updatedCheckpointSizes []uuid.UUID
		for i, c := range req.Checkpoints {
			if c.Resources != nil {
				size := int64(0)
				for _, v := range c.Resources.Resources {
					size += v
				}

				v2Update := tx.NewUpdate().Model(&model.CheckpointV2{}).
					Where("uuid = ?", c.Uuid)

				if len(c.Resources.Resources) == 0 { // Full delete case.
					v2Update = v2Update.Set("state = ?", model.DeletedState)
				} else { // Partial delete case.
					v2Update = v2Update.
						Set("resources = ?", c.Resources.Resources).
						Set("size = ?", size)

					oldResources := struct {
						bun.BaseModel `bun:"table:checkpoints_view"`
						Resources     map[string]int64
					}{}
					if err := tx.NewSelect().Model(&oldResources).
						Where("uuid = ?", c.Uuid).
						Scan(ctx); err != nil {
						return err
					}

					// Add metadata.json to oldResources if it is missing for backwards compatibility.
					_, alreadyHasMetadata := oldResources.Resources["metadata.json"]
					metadataValue, provided := c.Resources.Resources["metadata.json"]
					if !alreadyHasMetadata && provided {
						oldResources.Resources["metadata.json"] = metadataValue
					}

					// Only set state to partially deleted if files changed.
					if !maps.Equal(oldResources.Resources, c.Resources.Resources) {
						v2Update = v2Update.Set("state = ?", model.PartiallyDeletedState)
					}
				}

				if _, err := v2Update.Exec(ctx); err != nil {
					return fmt.Errorf("deleting checkpoints from checkpoints_v2: %w", err)
				}

				updatedCheckpointSizes = append(updatedCheckpointSizes, uuids[i])
			}
		}

		if len(updatedCheckpointSizes) > 0 {
			if err := db.UpdateCheckpointSizeTx(ctx, tx, updatedCheckpointSizes); err != nil {
				return fmt.Errorf("updating checkpoint size: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error patching checkpoints: %w", err)
	}

	return &apiv1.PatchCheckpointsResponse{}, nil
}

func (a *apiServer) CheckpointsRemoveFiles(
	ctx context.Context,
	req *apiv1.CheckpointsRemoveFilesRequest,
) (*apiv1.CheckpointsRemoveFilesResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	conv := &protoconverter.ProtoConverter{}
	checkpointsToDelete := conv.ToUUIDList(req.CheckpointUuids)
	if cErr := conv.Error(); cErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, "converting checkpoint: %s", cErr)
	}

	for _, g := range req.CheckpointGlobs {
		if len(g) == 0 {
			// Avoid weirdness where someone passes in "" then we concat {uuid}/{glob}
			// and we unexpectedly delete the whole checkpoint folder.
			return nil, status.Errorf(codes.InvalidArgument, "cannot have empty string glob")
		}
		if strings.Contains(g, "..") {
			return nil, status.Errorf(codes.InvalidArgument, "glob '%s' cannot contain '..'", g)
		}
	}

	exps, groupCUUIDsByEIDs, err := a.checkpointsRBACEditCheck(ctx, checkpointsToDelete)
	if err != nil {
		return nil, err
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

	taskSpec := *a.m.taskSpec

	var expIDs []int
	for _, g := range groupCUUIDsByEIDs {
		expIDs = append(expIDs, g.ExperimentID)
	}
	workspaceIDs, err := workspace.WorkspacesIDsByExperimentIDs(ctx, expIDs)
	if err != nil {
		return nil, err
	}

	jobID := model.NewJobID()
	if err = a.m.db.AddJob(&model.Job{
		JobID:   jobID,
		JobType: model.JobTypeCheckpointGC,
		OwnerID: &curUser.ID,
	}); err != nil {
		return nil, fmt.Errorf("persisting new job: %w", err)
	}

	// Submit checkpoint GC tasks for all checkpoints.
	for i, expIDcUUIDs := range groupCUUIDsByEIDs {
		i := i
		agentUserGroup, err := user.GetAgentUserGroup(ctx, curUser.ID, workspaceIDs[i])
		if err != nil {
			return nil, err
		}

		jobSubmissionTime := time.Now().UTC().Truncate(time.Millisecond)
		taskID := model.NewTaskID()
		conv := &protoconverter.ProtoConverter{}
		checkpointUUIDs := conv.ToUUIDList(strings.Split(expIDcUUIDs.CheckpointUUIDSStr, ","))

		go func() {
			err = runCheckpointGCTask(
				a.m.rm, a.m.db, taskID, jobID, jobSubmissionTime, taskSpec, exps[i].ID,
				exps[i].Config, checkpointUUIDs, req.CheckpointGlobs, false, agentUserGroup, curUser,
				nil,
			)
			if err != nil {
				log.WithError(err).Error("failed to start checkpoint GC task")
			}
		}()
	}

	return &apiv1.CheckpointsRemoveFilesResponse{}, nil
}

func (a *apiServer) DeleteCheckpoints(
	ctx context.Context,
	req *apiv1.DeleteCheckpointsRequest,
) (*apiv1.DeleteCheckpointsResponse, error) {
	if _, err := a.CheckpointsRemoveFiles(ctx, &apiv1.CheckpointsRemoveFilesRequest{
		CheckpointUuids: req.CheckpointUuids,
		CheckpointGlobs: []string{fullDeleteGlob},
	}); err != nil {
		return nil, err
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

	if err := a.m.canDoActionOnCheckpoint(ctx, *curUser, req.Checkpoint.Uuid,
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	currCheckpoint := &checkpointv1.Checkpoint{}
	if err := a.m.db.QueryProto("get_checkpoint", currCheckpoint, req.Checkpoint.Uuid); err != nil {
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

// Query for all trials that use a given checkpoint and return their metrics.
func (a *apiServer) GetTrialMetricsByCheckpoint(
	ctx context.Context, req *apiv1.GetTrialMetricsByCheckpointRequest,
) (*apiv1.GetTrialMetricsByCheckpointResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	err = a.m.canDoActionOnCheckpoint(ctx, *curUser, req.CheckpointUuid,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts)
	if err != nil {
		return nil, err
	}

	resp := &apiv1.GetTrialMetricsByCheckpointResponse{}
	trialIDsQuery := db.Bun().NewSelect().Table("trial_source_infos").
		Where("checkpoint_uuid = ?", req.CheckpointUuid)

	if req.TrialSourceInfoType != nil {
		trialIDsQuery.Where("trial_source_info_type = ?", req.TrialSourceInfoType.String())
	}

	metrics, err := trials.GetMetricsForTrialSourceInfoQuery(ctx, trialIDsQuery, req.MetricGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get trial source info %w", err)
	}

	resp.Metrics = append(resp.Metrics, metrics...)
	return resp, nil
}
