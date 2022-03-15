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
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

func (a *apiServer) GetProjectFromID(id int32) (*projectv1.Project, error) {
	p := &projectv1.Project{}
	switch err := a.m.db.QueryProto("get_project", p, id); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "project \"%d\" not found", id)
	default:
		return p, errors.Wrapf(err,
			"error fetching project \"%d\" from database", id)
	}
}

func (a *apiServer) GetProject(
	_ context.Context, req *apiv1.GetProjectRequest) (*apiv1.GetProjectResponse, error) {
	p, err := a.GetProjectFromID(req.Id)
	return &apiv1.GetProjectResponse{Project: p}, err
}

func (a *apiServer) GetProjectExperiments(
	_ context.Context, req *apiv1.GetProjectExperimentsRequest) (*apiv1.GetProjectExperimentsResponse, error) {
	// Verify that project exists.
	if _, err := a.GetProjectFromID(req.Id); err != nil {
		return nil, err
	}

	// Construct the experiment filtering expression.
	var allStates []string
	for _, state := range req.States {
		allStates = append(allStates, strings.TrimPrefix(state.String(), "STATE_"))
	}
	stateFilterExpr := strings.Join(allStates, ",")
	userFilterExpr := strings.Join(req.Users, ",")
	labelFilterExpr := strings.Join(req.Labels, ",")
	archivedExpr := ""
	if req.Archived != nil {
		archivedExpr = strconv.FormatBool(req.Archived.Value)
	}

	// Construct the ordering expression.
	orderColMap := map[apiv1.GetProjectExperimentsRequest_SortBy]string{
		apiv1.GetProjectExperimentsRequest_SORT_BY_UNSPECIFIED: "id",
		apiv1.GetProjectExperimentsRequest_SORT_BY_ID:          "id",
		apiv1.GetProjectExperimentsRequest_SORT_BY_DESCRIPTION: "description",
		apiv1.GetProjectExperimentsRequest_SORT_BY_NAME:        "name",
		apiv1.GetProjectExperimentsRequest_SORT_BY_START_TIME:  "start_time",
		apiv1.GetProjectExperimentsRequest_SORT_BY_END_TIME:    "end_time",
		apiv1.GetProjectExperimentsRequest_SORT_BY_STATE:       "state",
		apiv1.GetProjectExperimentsRequest_SORT_BY_NUM_TRIALS:  "num_trials",
		apiv1.GetProjectExperimentsRequest_SORT_BY_PROGRESS:    "COALESCE(progress, 0)",
		apiv1.GetProjectExperimentsRequest_SORT_BY_USER:        "username",
	}
	sortByMap := map[apiv1.OrderBy]string{
		apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
		apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
		apiv1.OrderBy_ORDER_BY_DESC:        "DESC NULLS LAST",
	}
	orderExpr := ""
	switch _, ok := orderColMap[req.SortBy]; {
	case !ok:
		return nil, fmt.Errorf("unsupported sort by %s", req.SortBy)
	case orderColMap[req.SortBy] != "id": //nolint:goconst // Not actually the same constant.
		orderExpr = fmt.Sprintf(
			"%s %s, id %s",
			orderColMap[req.SortBy], sortByMap[req.OrderBy], sortByMap[req.OrderBy],
		)
	default:
		orderExpr = fmt.Sprintf("id %s", sortByMap[req.OrderBy])
	}

	resp := &apiv1.GetProjectExperimentsResponse{}
	return resp, a.m.db.QueryProtof(
		"get_experiments",
		[]interface{}{orderExpr},
		resp,
		stateFilterExpr,
		archivedExpr,
		userFilterExpr,
		labelFilterExpr,
		req.Description,
		req.Name,
		req.Id,
		req.Offset,
		req.Limit,
	)
}

func (a *apiServer) PostProject(
	ctx context.Context, req *apiv1.PostProjectRequest) (*apiv1.PostProjectResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	p := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project", p, req.Name, req.Description, user.User.Id)

	return &apiv1.PostProjectResponse{Project: p},
		errors.Wrapf(err, "error creating project %s in database", req.Name)
}
