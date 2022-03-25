package ckpts

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

type CkptsAPI struct{}

func (s *CkptsAPI) GetCheckpoint(
	ctx context.Context, req *apiv1.GetCheckpointRequest) (*apiv1.GetCheckpointResponse, error) {
	ckpt, err := ByUUIDExpanded(ctx, req.CheckpointUuid)
	if err != nil {
		return nil, err
	}

	pc := protoutils.ProtoConverter{}
	protoCkpt := ckpt.ToProto(&pc)
	resp := &apiv1.GetCheckpointResponse{Checkpoint: &protoCkpt}
	return resp, pc.Error()
}

func (a *CkptsAPI) GetTrialCheckpoints(
	ctx context.Context, req *apiv1.GetTrialCheckpointsRequest,
) (*apiv1.GetTrialCheckpointsResponse, error) {
	// // XXX: check trial exists
	// switch exists, err := a.m.db.CheckTrialExists(int(req.Id)); {
	// case err != nil:
	// 	return nil, err
	// case !exists:
	// 	return nil, status.Error(codes.NotFound, "trial not found")
	// }

	var validationStates []string
	for _, vstate := range req.ValidationStates {
		switch vstate {
		case checkpointv1.State_STATE_ACTIVE:
			validationStates = append(validationStates, "'ACTIVE'")
		case checkpointv1.State_STATE_COMPLETED:
			validationStates = append(validationStates, "'COMPLETED'")
		case checkpointv1.State_STATE_ERROR:
			validationStates = append(validationStates, "'ERROR'")
		case checkpointv1.State_STATE_DELETED:
			validationStates = append(validationStates, "'DELETED'")
		default:
			// XXX: throw an error to the user
		}
	}

	// choose an ordering
	asc := req.OrderBy != apiv1.OrderBy_ORDER_BY_DESC
	var orderBy string
	switch req.SortBy {
	case apiv1.GetTrialCheckpointsRequest_SORT_BY_UNSPECIFIED:
		// noop, use the default
	case apiv1.GetTrialCheckpointsRequest_SORT_BY_BATCH_NUMBER:
		// noop, this is the default
	case apiv1.GetTrialCheckpointsRequest_SORT_BY_START_TIME:
		// noop, this field doesn't exist anymore
	case apiv1.GetTrialCheckpointsRequest_SORT_BY_END_TIME:
		orderBy = "report_time"
	case apiv1.GetTrialCheckpointsRequest_SORT_BY_VALIDATION_STATE:
		orderBy = "validation_metrics != 'null'::jsonb"
	case apiv1.GetTrialCheckpointsRequest_SORT_BY_STATE:
		orderBy = "state"
	}

	ckpts, total, err := GetTrialCheckpointsExpanded(
		ctx, int(req.Id), validationStates, orderBy, asc, int(req.Limit), int(req.Offset),
	)
	if err != nil {
		return nil, err
	}
	if len(ckpts) == 0 {
		// XXX: Why not return an empty array?
		return nil, status.Errorf(codes.NotFound, "no checkpoints found for trial %d", req.Id)
	}

	pc := protoutils.ProtoConverter{}
	resp := &apiv1.GetTrialCheckpointsResponse{}
	resp.Checkpoints = make([]*checkpointv1.Checkpoint, len(ckpts))
	for i, ckpt := range ckpts {
		protoCkpt := ckpt.ToProto(&pc)
		// XXX taking address of protoCkpt looks buggy to me
		resp.Checkpoints[i] = &protoCkpt
	}
	resp.Pagination = &apiv1.Pagination{
		Offset: req.Offset,
		Limit:  req.Limit,
		// XXX  Why have both offset and start_idx?
		StartIndex: req.Offset,
		EndIndex:   pc.ToInt32(int(req.Offset) + len(ckpts)),
		Total:      pc.ToInt32(total),
	}

	return resp, pc.Error()
}
