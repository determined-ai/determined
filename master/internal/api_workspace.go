package internal

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func (a *apiServer) GetWorkspace(
	_ context.Context, req *apiv1.GetWorkspaceRequest) (*apiv1.GetWorkspaceResponse, error) {
	w := &workspacev1.Workspace{}
	switch err := a.m.db.QueryProto("get_workspace", w, req.Id); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "workspace \"%d\" not found", req.Id)
	default:
		return &apiv1.GetWorkspaceResponse{Workspace: w},
			errors.Wrapf(err, "error fetching workspace \"%d\" from database", req.Id)
	}
}

func (a *apiServer) GetWorkspaces(
	_ context.Context, req *apiv1.GetWorkspacesRequest) (*apiv1.GetWorkspacesResponse, error) {
	resp := &apiv1.GetWorkspacesResponse{}
	nameFilter := req.Name
	archFilterExpr := ""
	if req.Archived != nil {
		archFilterExpr = strconv.FormatBool(req.Archived.Value)
	}
	userFilterExpr := strings.Join(req.Users, ",")
	// Construct the ordering expression.
	sortColMap := map[apiv1.GetWorkspacesRequest_SortBy]string{
		apiv1.GetWorkspacesRequest_SORT_BY_UNSPECIFIED:       "id",
		apiv1.GetWorkspacesRequest_SORT_BY_NAME:              "name",
	}
	orderByMap := map[apiv1.OrderBy]string{
		apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
		apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
		apiv1.OrderBy_ORDER_BY_DESC:        "DESC",
	}
	orderExpr := ""
	switch _, ok := sortColMap[req.SortBy]; {
	case !ok:
		return nil, fmt.Errorf("unsupported sort by %s", req.SortBy)
	case sortColMap[req.SortBy] != "id":
		orderExpr = fmt.Sprintf(
			"%s %s, id %s",
			sortColMap[req.SortBy], orderByMap[req.OrderBy], orderByMap[req.OrderBy],
		)
	default:
		orderExpr = fmt.Sprintf("id %s", orderByMap[req.OrderBy])
	}
	err := a.m.db.QueryProtof(
		"get_workspaces",
		[]interface{}{orderExpr},
		&resp.Workspaces,
		userFilterExpr,
		nameFilter,
		archFilterExpr,
	)
	if err != nil {
		return nil, err
	}
	return resp, a.paginate(&resp.Pagination, &resp.Workspaces, req.Offset, req.Limit)
}
