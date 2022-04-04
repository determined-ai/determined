package checkpoints

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CheckpointServer struct {
	// TODO: Remove this DB eventually
	db *db.PgDB
}

func NewCheckpointServer(db *db.PgDB) *CheckpointServer {
	return &CheckpointServer{db: db}
}

func (s *CheckpointServer) GetCheckpoint(
	ctx context.Context, req *apiv1.GetCheckpointRequest,
) (*apiv1.GetCheckpointResponse, error) {
	conv := protoconverter.ProtoConverter{}
	uuid := conv.ToUUID(req.CheckpointUuid)
	if err := conv.Error(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "converting uuid: %s", err)
	}

	switch c, err := ByUUID(ctx, uuid); err {
	case nil:
		pc, err := c.ToProto()
		if err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument, "converting checkpoint to proto: %s", err)
		}
		return &apiv1.GetCheckpointResponse{Checkpoint: pc}, nil
	case db.ErrNotFound, sql.ErrNoRows:
		return nil, status.Errorf(
			codes.NotFound, "checkpoint %s not found", uuid)
	default:
		return nil, status.Errorf(
			codes.Internal, "fetching checkpoint %s from database: %s", uuid, err)
	}
}

func (s *CheckpointServer) GetExperimentCheckpoints(
	ctx context.Context, req *apiv1.GetExperimentCheckpointsRequest,
) (*apiv1.GetExperimentCheckpointsResponse, error) {
	ok, err := s.db.CheckExperimentExists(int(req.Id))
	switch {
	case err != nil:
		return nil, status.Errorf(codes.Internal, "checking if the experiment exists: %s", err)
	case !ok:
		return nil, status.Errorf(codes.NotFound, "experiment %d not found", req.Id)
	}

	orderByColumn := ""
	switch req.SortBy {
	case apiv1.GetExperimentCheckpointsRequest_SORT_BY_TRIAL_ID:
		orderByColumn = "trial_id"
	case apiv1.GetExperimentCheckpointsRequest_SORT_BY_UUID:
		orderByColumn = "uuid"
	case apiv1.GetExperimentCheckpointsRequest_SORT_BY_BATCH_NUMBER:
		orderByColumn = "metadata->>'latest_batch'"
	case apiv1.GetExperimentCheckpointsRequest_SORT_BY_STATE:
		orderByColumn = "state"
	case apiv1.GetExperimentCheckpointsRequest_SORT_BY_END_TIME:
		orderByColumn = "report_time"
	case apiv1.GetExperimentCheckpointsRequest_SORT_BY_SEARCHER_METRIC:
		orderByColumn = "searcher_metric"
	case apiv1.GetExperimentCheckpointsRequest_SORT_BY_UNSPECIFIED:
		orderByColumn = "report time"
	default:
		return nil, status.Errorf(codes.InvalidArgument, "interpretting sort by: %s", req.SortBy)
	}

	orderByDirection, err := apiOrderByToSQL(req.OrderBy)
	if err != nil {
		return nil, err
	}

	var pageInfo db.PageInfo
	cs, err := List(ctx, func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
		q = q.Where("experiment_id = ?", req.Id)

		if len(req.States) > 0 {
			q = q.Where("state IN (?)", bun.In(req.States))
		}

		q, pageInfo, err = db.AddPagination(ctx, q, int(req.Offset), int(req.Limit))
		if err != nil {
			return nil, fmt.Errorf("converting offset: %w", err)
		}

		q = q.OrderExpr(fmt.Sprintf("%s %s, trial_id DESC", orderByColumn, orderByDirection))
		return q, nil
	})

	pcs := []*checkpointv1.CheckpointMetadata{}
	for _, c := range cs {
		pc, err := c.ToProto()
		if err != nil {
			return nil, fmt.Errorf("converting checkpoints to proto: %w", err)
		}
		pcs = append(pcs, pc)
	}

	return &apiv1.GetExperimentCheckpointsResponse{
		Checkpoints: pcs,
		Pagination:  pageInfo.ToProto(),
	}, nil
}

func (s *CheckpointServer) GetTrialCheckpoints(
	ctx context.Context, req *apiv1.GetTrialCheckpointsRequest,
) (*apiv1.GetTrialCheckpointsResponse, error) {
	switch ok, err := s.db.CheckTrialExists(int(req.Id)); {
	case err != nil:
		return nil, status.Errorf(codes.Internal, "checking if the trial exists: %s", err)
	case !ok:
		return nil, status.Error(codes.NotFound, "trial not found")
	}

	orderByColumn := ""
	switch req.SortBy {
	case apiv1.GetTrialCheckpointsRequest_SORT_BY_UNSPECIFIED:
		orderByColumn = "report_time"
	case apiv1.GetTrialCheckpointsRequest_SORT_BY_UUID:
		orderByColumn = "uuid"
	case apiv1.GetTrialCheckpointsRequest_SORT_BY_BATCH_NUMBER:
		orderByColumn = "metadata->>'latest_batch'"
	case apiv1.GetTrialCheckpointsRequest_SORT_BY_END_TIME:
		orderByColumn = "report_time"
	case apiv1.GetTrialCheckpointsRequest_SORT_BY_STATE:
		orderByColumn = "state"
	default:
		return nil, status.Errorf(codes.InvalidArgument, "interpretting sort by: %s", req.SortBy)
	}

	orderByDirection, err := apiOrderByToSQL(req.OrderBy)
	if err != nil {
		return nil, err
	}

	var pageInfo db.PageInfo
	cs, err := List(ctx, func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
		q = q.Where("trial_id = ?", req.Id)

		if len(req.States) > 0 {
			q = q.Where("state IN (?)", bun.In(req.States))
		}

		q, pageInfo, err = db.AddPagination(ctx, q, int(req.Offset), int(req.Limit))
		if err != nil {
			return nil, fmt.Errorf("converting offset: %w", err)
		}

		q = q.OrderExpr(fmt.Sprintf(
			"%s %s, metadata->>'latest_batch' DESC", orderByColumn, orderByDirection))
		return q, nil
	})

	pcs := []*checkpointv1.CheckpointMetadata{}
	for _, c := range cs {
		pc, err := c.ToProto()
		if err != nil {
			return nil, fmt.Errorf("converting checkpoints to proto: %w", err)
		}
		pcs = append(pcs, pc)
	}

	return &apiv1.GetTrialCheckpointsResponse{
		Checkpoints: pcs,
		Pagination:  pageInfo.ToProto(),
	}, nil
}

func (s *CheckpointServer) PostCheckpointMetadata(
	ctx context.Context, req *apiv1.PostCheckpointMetadataRequest,
) (*apiv1.PostCheckpointMetadataResponse, error) {
	conv := protoconverter.ProtoConverter{}
	uuid := conv.ToUUID(req.Checkpoint.Uuid)
	if err := conv.Error(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parsing uuid: %s", err)
	}

	c, err := ByUUID(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("fetching checkpoint %s from database: %w", req.Checkpoint.Uuid, err)
	}

	c.Metadata = req.Checkpoint.Metadata.AsMap()
	if err := c.Upsert(ctx); err != nil {
		return nil, fmt.Errorf("updating checkpoint %s in database: %w", uuid, err)
	}

	pc, err := c.ToProto()
	if err != nil {
		return nil, fmt.Errorf("converting checkpoint %s to proto: %w", req.Checkpoint.Uuid, err)
	}
	return &apiv1.PostCheckpointMetadataResponse{Checkpoint: pc}, nil
}

func apiOrderByToSQL(orderBy apiv1.OrderBy) (string, error) {
	switch orderBy {
	case apiv1.OrderBy_ORDER_BY_ASC:
		return "ASC", nil
	case apiv1.OrderBy_ORDER_BY_DESC:
		return "DESC", nil
	case apiv1.OrderBy_ORDER_BY_UNSPECIFIED:
		return "ASC", nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "interpretting order by: %s", orderBy)
	}
}
