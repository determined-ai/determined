package trials

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	exputil "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// TrialsAPIServer is an embedded api server struct.
type TrialsAPIServer struct{}

// StartTrial is called on Core API context enter in detached mode.
func (a *TrialsAPIServer) StartTrial(
	ctx context.Context, req *apiv1.StartTrialRequest,
) (*apiv1.StartTrialResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	trialID := int(req.TrialId)
	exp, err := db.ExperimentByTrialID(ctx, trialID)
	if err != nil {
		return nil, fmt.Errorf("failed to find experiment by trial id: %w", err)
	}

	if err = exputil.AuthZProvider.Get().CanEditExperimentsMetadata(
		ctx, *curUser, exp); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	if !exp.Unmanaged {
		return nil, errors.New("only unmanaged trials are supported")
	}

	obj := Trial{ID: trialID}

	err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if !req.Resume {
			err := tx.NewSelect().Model(&obj).WherePK().
				Column("run_id", "state").
				For("UPDATE").
				Scan(ctx, &obj)
			if err != nil {
				return err
			}
			if obj.RunID != 0 {
				return fmt.Errorf("trial has already been started")
			}
		}

		_, err := tx.NewUpdate().Model(&obj).WherePK().
			Set("run_id = run_id + 1").
			Set("state = ?", model.RunningState).
			Set("last_activity = ?", time.Now()).
			Returning("run_id").
			Exec(ctx)
		if err != nil {
			return err
		}

		return UpdateUnmanagedExperimentStatesTx(ctx, tx, []*model.Experiment{exp})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start trial: %w", err)
	}

	var stepsCompleted int
	var latestCheckpointUUID *string
	if obj.RunID > 1 {
		latestCheckpoint, err := LatestCheckpointForTrialTx(ctx, db.Bun(), trialID)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, fmt.Errorf("failed to find checkpoint for trial: %w", err)
		}

		if latestCheckpoint != nil {
			latestCheckpointUUID = ptrs.Ptr(latestCheckpoint.UUID.String())
			stepsCompleted = latestCheckpoint.StepsCompleted
		}
	}

	resp := &apiv1.StartTrialResponse{
		TrialRunId:       int32(obj.RunID),
		LatestCheckpoint: latestCheckpointUUID,
		StepsCompleted:   int32(stepsCompleted),
	}

	return resp, nil
}
