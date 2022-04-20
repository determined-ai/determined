package projects

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

// ProjectServer is a wrapper for project routes.
type ProjectServer struct {
	db     *db.PgDB
}

// NewProjectServer is a helper to create ProjectServer, a project route wrapper.
func NewProjectServer(db *db.PgDB) *ProjectServer {
	return &ProjectServer{db: db}
}

const byname = "name"
const bydescription = "description"

// GetProject is a request for information about a specific project by its ID.
func (s *ProjectServer) GetProject(
	ctx context.Context, req *apiv1.GetProjectRequest,
) (*apiv1.GetProjectResponse, error) {
	switch p, err := ByID(ctx, req.Id); err {
	case nil:
		pc, err2 := p.ToProto()
		if err2 != nil {
			return nil, status.Errorf(
				codes.InvalidArgument, "converting project to proto: %s", err2)
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

// GetWorkspaceProjects is a request for information about all project in a workspace by its ID.
func (s *ProjectServer) GetWorkspaceProjects(
	ctx context.Context, req *apiv1.GetWorkspaceProjectsRequest,
) (*apiv1.GetWorkspaceProjectsResponse, error) {
	const byid = "project_metadata.id"
	orderByColumn := ""
	switch req.SortBy {
	case apiv1.GetWorkspaceProjectsRequest_SORT_BY_CREATION_TIME:
		orderByColumn = "created_at"
	case apiv1.GetWorkspaceProjectsRequest_SORT_BY_LAST_EXPERIMENT_START_TIME:
		orderByColumn = "last_experiment_started_at"
	case apiv1.GetWorkspaceProjectsRequest_SORT_BY_ID:
		orderByColumn = byid
	case apiv1.GetWorkspaceProjectsRequest_SORT_BY_NAME:
		orderByColumn = byname
	case apiv1.GetWorkspaceProjectsRequest_SORT_BY_DESCRIPTION:
		orderByColumn = bydescription
	case apiv1.GetWorkspaceProjectsRequest_SORT_BY_UNSPECIFIED:
		orderByColumn = byid
	default:
		return nil, status.Errorf(codes.InvalidArgument, "interpreting sort by: %s", req.SortBy)
	}

	orderByDirection, err := apiOrderByToSQL(req.OrderBy)
	if err != nil {
		return nil, err
	}

	var pageInfo db.PageInfo
	cs, err := ProjectList(ctx, func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
		q = q.Where("workspace_id = ?", req.Id)

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

	pcs := []*projectv1.Project{}
	for _, c := range cs {
		pc, err := c.ToProto()
		if err != nil {
			return nil, fmt.Errorf("converting projects to proto: %w", err)
		}
		pcs = append(pcs, pc)
	}

	return &apiv1.GetWorkspaceProjectsResponse{
		Projects:   pcs,
		Pagination: pageInfo.ToProto(),
	}, nil
}

// OwnedWorkspaces returns a query for IDs of workspaces which were created by this user.
func ownedWorkspaces(user_id int32) (*bun.SelectQuery) {
	return db.Bun().NewSelect().
		ColumnExpr("id").
		TableExpr("workspaces").
		Where("NOT immutable").
		Where("user_id = ?", user_id)
}

// DeleteProject is a request to delete a project by ID.
func (s *ProjectServer) DeleteProject(ctx context.Context,
	req *apiv1.DeleteProjectRequest) (*apiv1.DeleteProjectResponse, error) {
	user, _, err := grpcutil.GetUser(ctx, s.db, &s.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}

	owned_workspaces := ownedWorkspaces(int32(user.ID))
	err = ProjectDeletion(ctx, func(q *bun.DeleteQuery) (*bun.DeleteQuery, error) {
		q = q.Where("id = ?", req.Id)
		q = q.Where("user_id = ? OR ? IS TRUE OR workspace_id IN (?)",
			user.ID, user.Admin, owned_workspaces,
		)
		return q, nil
	})
	return &apiv1.DeleteProjectResponse{}, err
}

// GetProjectExperiments is a request for information about all experiments in a project by its ID.
func (s *ProjectServer) GetProjectExperiments(ctx context.Context,
	req *apiv1.GetProjectExperimentsRequest) (*apiv1.GetProjectExperimentsResponse,
	error) {
	// Verify that project exists.
	_, err := ByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	// Construct the ordering expression.
	const byid = "experiment_metadata.id"
	orderByColumn := ""
	switch req.SortBy {
	case apiv1.GetProjectExperimentsRequest_SORT_BY_START_TIME:
		orderByColumn = "start_time"
	case apiv1.GetProjectExperimentsRequest_SORT_BY_END_TIME:
		orderByColumn = "end_time"
	case apiv1.GetProjectExperimentsRequest_SORT_BY_ID:
		orderByColumn = byid
	case apiv1.GetProjectExperimentsRequest_SORT_BY_NAME:
		orderByColumn = byname
	case apiv1.GetProjectExperimentsRequest_SORT_BY_DESCRIPTION:
		orderByColumn = bydescription
	case apiv1.GetProjectExperimentsRequest_SORT_BY_STATE:
		orderByColumn = "state"
	case apiv1.GetProjectExperimentsRequest_SORT_BY_NUM_TRIALS:
		orderByColumn = "num_trials"
	case apiv1.GetProjectExperimentsRequest_SORT_BY_PROGRESS:
		orderByColumn = "COALESCE(progress, 0)"
	case apiv1.GetProjectExperimentsRequest_SORT_BY_USER:
		orderByColumn = "username"
	case apiv1.GetProjectExperimentsRequest_SORT_BY_UNSPECIFIED:
		orderByColumn = byid
	default:
		return nil, status.Errorf(codes.InvalidArgument, "interpreting sort by: %s", req.SortBy)
	}

	orderByDirection, err := apiOrderByToSQL(req.OrderBy)
	if err != nil {
		return nil, err
	}

	var pageInfo db.PageInfo
	resp, err := ExperimentList(ctx, func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
		q = q.Where("project_id = ?", req.Id)

		if req.Archived != nil {
			q = q.Where("archived = ?", req.Archived.Value)
		}
		if req.Name != "" {
			q = q.Where("name ILIKE ?", "%"+req.Name+"%")
		}
		if req.Description != "" {
			q = q.Where("description ILIKE ?", "%"+req.Description+"%")
		}
		if req.Labels != nil {
			q = q.Where("labels = ?", req.Labels)
		}
		if req.States != nil {
			var allStates []string
			for _, state := range req.States {
				allStates = append(allStates, strings.TrimPrefix(state.String(), "STATE_"))
			}
			q = q.Where("state IN (?)", bun.In(allStates))
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

	exps := []*experimentv1.Experiment{}
	for _, exp := range resp {
		pexp, err := exp.ToProto()
		if err != nil {
			return nil, fmt.Errorf("converting experiments to proto: %w", err)
		}
		exps = append(exps, pexp)
	}

	return &apiv1.GetProjectExperimentsResponse{
		Experiments: exps,
		Pagination:  pageInfo.ToProto(),
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
