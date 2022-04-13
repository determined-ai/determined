package projects

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProjectServer struct {
	// TODO: Remove this DB eventually (??)
	db *db.PgDB
}

func NewProjectServer(db *db.PgDB) *ProjectServer {
	return &ProjectServer{db: db}
}

func (s *ProjectServer) GetProject(
	ctx context.Context, req *apiv1.GetProjectRequest,
) (*apiv1.GetProjectResponse, error) {
	switch p, err := ByID(ctx, req.Id); err {
	case nil:
		pc, err := p.ToProto()
		if err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument, "converting project to proto: %s", err)
		}
		return &apiv1.GetProjectResponse{Project: pc}, nil
	case db.ErrNotFound, sql.ErrNoRows:
		return nil, status.Errorf(
			codes.NotFound, "project %v not found", req.Id)
	default:
		return nil, status.Errorf(
			codes.Internal, "fetching project %v from database: %s", req.Id, err)
	}
}

func (s *ProjectServer) GetWorkspaceProjects(
	ctx context.Context, req *apiv1.GetWorkspaceProjectsRequest,
) (*apiv1.GetWorkspaceProjectsResponse, error) {
  orderByColumn := ""
	switch req.SortBy {
	case apiv1.GetWorkspaceProjectsRequest_SORT_BY_CREATION_TIME:
		orderByColumn = "created_at"
  case apiv1.GetWorkspaceProjectsRequest_SORT_BY_LAST_EXPERIMENT_START_TIME:
		orderByColumn = "last_experiment_started_at"
  case apiv1.GetWorkspaceProjectsRequest_SORT_BY_ID:
		orderByColumn = "id"
  case apiv1.GetWorkspaceProjectsRequest_SORT_BY_NAME:
		orderByColumn = "name"
  case apiv1.GetWorkspaceProjectsRequest_SORT_BY_DESCRIPTION:
		orderByColumn = "description"
	case apiv1.GetWorkspaceProjectsRequest_SORT_BY_UNSPECIFIED:
		orderByColumn = "id"
	default:
		return nil, status.Errorf(codes.InvalidArgument, "interpreting sort by: %s", req.SortBy)
	}

	orderByDirection, err := apiOrderByToSQL(req.OrderBy)
	if err != nil {
		return nil, err
	}

	var pageInfo db.PageInfo
	cs, err := List(ctx, func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
		q = q.Where("workspace_id = ?", req.Id)

    if req.Archived != nil {
      q = q.Where("archived = ?", req.Archived.Value)
    }
    if req.Name != "" {
      q = q.Where("name ILIKE ?", "%" + req.Name + "%")
    }
    if len(req.Users) > 0 {
    	q = q.Where("username IN (?)", bun.In(req.Users))
    }

		q, pageInfo, err = db.AddPagination(ctx, q, int(req.Offset), int(req.Limit))
		if err != nil {
			return nil, fmt.Errorf("converting offset: %w", err)
		}

		q = q.OrderExpr(fmt.Sprintf("%s %s", orderByColumn, orderByDirection))
		return q, nil
	})

	pcs := []*projectv1.Project{}
	for _, c := range cs {
		pc, err := c.ToProto()
		if err != nil {
			return nil, fmt.Errorf("converting projects to proto: %w", err)
		}
		pcs = append(pcs, pc)
	}

	return &apiv1.GetWorkspaceProjectsResponse{
		Projects:    pcs,
		Pagination:  pageInfo.ToProto(),
	}, nil
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
		return "", status.Errorf(codes.InvalidArgument, "interpreting order by: %s", orderBy)
	}
}
