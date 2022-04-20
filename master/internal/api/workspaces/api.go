package workspaces

import (
	"context"
	"fmt"
	"database/sql"
	
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// WorkspaceServer is a wrapper for workspace routes.
type WorkspaceServer struct {
	db *db.PgDB
}

// NewWorkspaceServer is a helper to create WorkspaceServer, a workspace route wrapper.
func NewWorkspaceServer(db *db.PgDB) *WorkspaceServer {
	return &WorkspaceServer{db: db}
}

// GetWorkspace is a request for information about a specific workspace by its ID.
func (s *WorkspaceServer) GetWorkspace(
	ctx context.Context, req *apiv1.GetWorkspaceRequest,
) (*apiv1.GetWorkspaceResponse, error) {
	switch p, err := ByID(ctx, req.Id); err {
	case nil:
		pc, err2 := p.ToProto()
		if err2 != nil {
			return nil, status.Errorf(
				codes.InvalidArgument, "converting workspace to proto: %s", err2)
		}
		return &apiv1.GetWorkspaceResponse{Workspace: pc}, nil
	case db.ErrNotFound, sql.ErrNoRows:
		return nil, status.Errorf(
			codes.NotFound, "workspace %v not found", req.Id)
	default:
		return nil, status.Errorf(
			codes.Internal, "fetching workspace %v from database: %s", req.Id, err)
	}
}

// GetWorkspaces is a request for information about all workspaces matching search criteria.
func (s *WorkspaceServer) GetWorkspaces(
	ctx context.Context, req *apiv1.GetWorkspacesRequest) (*apiv1.GetWorkspacesResponse, error) {

	const byid = "id"
	orderByColumn := ""
	switch req.SortBy {
	case apiv1.GetWorkspacesRequest_SORT_BY_ID:
		orderByColumn = byid
	case apiv1.GetWorkspacesRequest_SORT_BY_NAME:
		orderByColumn = "name"
	case apiv1.GetWorkspacesRequest_SORT_BY_UNSPECIFIED:
		orderByColumn = byid
	default:
		return nil, status.Errorf(codes.InvalidArgument, "interpreting sort by: %s", req.SortBy)
	}

	orderByDirection, err := apiOrderByToSQL(req.OrderBy)
	if err != nil {
		return nil, err
	}

	var pageInfo db.PageInfo
	cs, err := WorkspaceList(ctx, func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
		if req.Archived != nil {
			q = q.Where("archived = ?", req.Archived.Value)
		}
		if req.Name != "" {
			q = q.Where("name ILIKE ?", "%"+req.Name+"%")
		}
		if len(req.Users) > 0 {
			q = q.Where("username IN (?)", bun.In(req.Users))
		}
		if req.Limit > 0 {
			q = q.Limit(int(req.Limit))
		}
		if req.Offset > 0 {
			q = q.Offset(int(req.Offset))
		}

		q, pageInfo, err = db.AddPagination(ctx, q, int(req.Offset), int(req.Limit))
		if err != nil {
			return nil, fmt.Errorf("converting offset: %w", err)
		}

		q = q.OrderExpr(fmt.Sprintf("%s %s", orderByColumn, orderByDirection))
		return q, nil
	})

	pcs := []*workspacev1.Workspace{}
	for _, c := range cs {
		pc, err := c.ToProto()
		if err != nil {
			return nil, fmt.Errorf("converting workspaces to proto: %w", err)
		}
		pcs = append(pcs, pc)
	}

	return &apiv1.GetWorkspacesResponse{
		Workspaces: pcs,
		Pagination: pageInfo.ToProto(),
	}, nil
}

func apiOrderByToSQL(orderBy apiv1.OrderBy) (string, error) {
	const asc = "ASC"
	switch orderBy {
	case apiv1.OrderBy_ORDER_BY_ASC:
		return asc, nil
	case apiv1.OrderBy_ORDER_BY_DESC:
		return "DESC", nil
	case apiv1.OrderBy_ORDER_BY_UNSPECIFIED:
		return asc, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "interpreting order by: %s", orderBy)
	}
}
