package internal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slices"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/project"
	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/hpimportance"
	"github.com/determined-ai/determined/master/internal/lttb"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoless"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"

	structpb "github.com/golang/protobuf/ptypes/struct"
)

var experimentsAddr = actor.Addr("experiments")

// Catches information on active running experiments.
type experimentAllocation struct {
	Job      model.JobID
	Pulling  bool
	Running  bool
	Starting bool
}

// Enrich one or more experiments by converting Active state to Queued/Pulling/Starting/Running.
func (a *apiServer) enrichExperimentState(experiments ...*experimentv1.Experiment) error {
	// filter allocations by JobIDs on this page of experiments
	jobFilter := make([]string, 0, len(experiments))
	for _, exp := range experiments {
		jobFilter = append(jobFilter, exp.JobId)
	}

	// get active experiments by JobID
	tasks := []experimentAllocation{}
	err := a.m.db.Query(
		"aggregate_allocation_state_by_job",
		&tasks,
		strings.Join(jobFilter, ","),
	)
	if err != nil {
		return err
	}

	// Collect state information by JobID
	byJobID := make(map[model.JobID]experimentv1.State, len(tasks))
	for _, task := range tasks {
		switch {
		case task.Running:
			byJobID[task.Job] = experimentv1.State_STATE_RUNNING
		case task.Starting:
			byJobID[task.Job] = model.MostProgressedExperimentState(byJobID[task.Job],
				experimentv1.State_STATE_STARTING)
		case task.Pulling:
			byJobID[task.Job] = model.MostProgressedExperimentState(byJobID[task.Job],
				experimentv1.State_STATE_PULLING)
		default:
			byJobID[task.Job] = model.MostProgressedExperimentState(byJobID[task.Job],
				experimentv1.State_STATE_QUEUED)
		}
	}

	// Active experiments converted to Queued, Pulling, Starting, or Running
	for _, exp := range experiments {
		if exp.State == experimentv1.State_STATE_ACTIVE {
			if setState, ok := byJobID[model.JobID(exp.JobId)]; ok {
				exp.State = setState
			} else {
				exp.State = experimentv1.State_STATE_QUEUED
			}
		}
	}
	return nil
}

// Return if experiment state is Active or any of its sub-states.
func isActiveExperimentState(state experimentv1.State) bool {
	return slices.Contains([]experimentv1.State{
		experimentv1.State_STATE_ACTIVE,
		experimentv1.State_STATE_PULLING, experimentv1.State_STATE_QUEUED,
		experimentv1.State_STATE_RUNNING, experimentv1.State_STATE_STARTING,
	}, state)
}

// Return a single experiment with enriched state, if the user can access it.
func (a *apiServer) getExperiment(
	ctx context.Context, curUser model.User, experimentID int,
) (*experimentv1.Experiment, error) {
	expNotFound := status.Errorf(codes.NotFound, "experiment not found: %d", experimentID)
	exp := &experimentv1.Experiment{}
	if err := a.m.db.QueryProto("get_experiment", exp, experimentID); errors.Is(err, db.ErrNotFound) {
		return nil, expNotFound
	} else if err != nil {
		return nil, errors.Wrapf(err, "error fetching experiment from database: %d", experimentID)
	}

	modelExp, err := model.ExperimentFromProto(exp)
	if err != nil {
		return nil, err
	}
	if ok, authErr := expauth.AuthZProvider.Get().
		CanGetExperiment(ctx, curUser, modelExp); authErr != nil {
		return nil, authErr
	} else if !ok {
		return nil, expNotFound
	}

	sort.Slice(exp.TrialIds, func(i, j int) bool {
		return exp.TrialIds[i] < exp.TrialIds[j]
	})

	if err = a.enrichExperimentState(exp); err != nil {
		return nil, err
	}

	return exp, nil
}

func (a *apiServer) getExperimentAndCheckCanDoActions(
	ctx context.Context,
	expID int,
	withConfig bool,
	actions ...func(context.Context, model.User, *model.Experiment) error,
) (*model.Experiment, model.User, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, model.User{}, err
	}

	var e *model.Experiment
	if withConfig {
		e, err = a.m.db.ExperimentByID(expID)
	} else {
		e, err = a.m.db.ExperimentWithoutConfigByID(expID)
	}
	expNotFound := status.Errorf(codes.NotFound, "experiment not found: %d", expID)
	if errors.Is(err, db.ErrNotFound) {
		return nil, model.User{}, expNotFound
	} else if err != nil {
		return nil, model.User{}, err
	}

	var ok bool
	if ok, err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, e); err != nil {
		return nil, model.User{}, err
	} else if !ok {
		return nil, model.User{}, expNotFound
	}

	for _, action := range actions {
		if err = action(ctx, *curUser, e); err != nil {
			return nil, model.User{}, status.Errorf(codes.PermissionDenied, err.Error())
		}
	}
	return e, *curUser, nil
}

func (a *apiServer) GetSearcherEvents(
	ctx context.Context, req *apiv1.GetSearcherEventsRequest,
) (*apiv1.GetSearcherEventsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	exp, err := a.getExperiment(ctx, *curUser, int(req.ExperimentId))
	if err != nil {
		return nil, err
	}
	if !isActiveExperimentState(exp.State) {
		return &apiv1.GetSearcherEventsResponse{
			SearcherEvents: []*experimentv1.SearcherEvent{{
				Id: -1,
				Event: &experimentv1.SearcherEvent_ExperimentInactive{
					ExperimentInactive: &experimentv1.ExperimentInactive{
						ExperimentState: exp.State,
					},
				},
			}},
		}, nil
	}

	addr := experimentsAddr.Child(req.ExperimentId)
	var w searcher.EventsWatcher
	if err = a.ask(addr, req, &w); err != nil {
		return nil, status.Errorf(codes.Internal,
			"failed to get events from actor: long polling %v", err)
	}

	defer a.m.system.TellAt(addr, UnwatchEvents{w.ID})

	ctx, cancel := context.WithTimeout(ctx, time.Duration(60)*time.Second)
	defer cancel()

	select {
	case events := <-w.C:
		return &apiv1.GetSearcherEventsResponse{
			SearcherEvents: events,
		}, nil
	case <-ctx.Done():
		return &apiv1.GetSearcherEventsResponse{
			SearcherEvents: nil,
		}, nil
	}
}

func (a *apiServer) PostSearcherOperations(
	ctx context.Context,
	req *apiv1.PostSearcherOperationsRequest,
) (
	resp *apiv1.PostSearcherOperationsResponse, err error,
) {
	_, _, err = a.getExperimentAndCheckCanDoActions(
		ctx, int(req.ExperimentId), false, expauth.AuthZProvider.Get().CanRunCustomSearch,
	)
	if err != nil {
		return nil, errors.Wrap(err, "fetching experiment from database")
	}

	addr := experimentsAddr.Child(req.ExperimentId)
	switch err = a.ask(addr, req, &resp); {
	case err != nil:
		return nil, status.Errorf(codes.Internal, "failed to post operations: %v", err)
	default:
		logrus.Infof("posted operations %v", req.SearcherOperations)
		return resp, nil
	}
}

func (a *apiServer) GetExperiment(
	ctx context.Context, req *apiv1.GetExperimentRequest,
) (*apiv1.GetExperimentResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}
	exp, err := a.getExperiment(ctx, *user, int(req.ExperimentId))
	if err != nil {
		return nil, err
	}

	resp := apiv1.GetExperimentResponse{
		Experiment: exp,
	}

	// Only continue to add a job summary if it's an active experiment.
	if !isActiveExperimentState(exp.State) {
		return &resp, nil
	}

	jobID := model.JobID(exp.JobId)

	jobSummary := &jobv1.JobSummary{}
	err = a.ask(sproto.JobsActorAddr, sproto.GetJobSummary{
		JobID:        jobID,
		ResourcePool: exp.ResourcePool,
	}, &jobSummary)
	if err != nil {
		// An error here either is real or just that the experiment was not yet terminal in the DB
		// when we first queried it but was by the time it got around to handling out ask. We can't
		// just refresh our DB state to see which it was, since there is a time between an actor
		// closing and PostStop (where the DB state is set) being received where the actor may not
		// respond but still is not terminal -- more clearly, there is a time where the actor is
		// truly non-terminal and not reachable. We _could_ await its stop and recheck, but it's not
		// easy deducible how long that would block. So the best we can really do is return without
		// an error if we're in this case and log. This is a debug log because of how often the
		// happens when polling for an experiment to end.
		if !strings.Contains(err.Error(), sproto.ErrJobNotFound(jobID).Error()) {
			return nil, err
		}
		logrus.WithError(err).Debugf("asking for job summary")
	} else {
		resp.JobSummary = jobSummary
	}

	return &resp, nil
}

func (a *apiServer) DeleteExperiment(
	ctx context.Context, req *apiv1.DeleteExperimentRequest,
) (*apiv1.DeleteExperimentResponse, error) {
	e, curUser, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId), false,
		expauth.AuthZProvider.Get().CanDeleteExperiment)
	if err != nil {
		return nil, err
	}

	switch exists, eErr := a.m.db.ExperimentHasCheckpointsInRegistry(int(req.ExperimentId)); {
	case eErr != nil:
		return nil, errors.New("failed to check model registry for references")
	case exists:
		return nil, status.Errorf(
			codes.InvalidArgument, "checkpoints are registered as model versions")
	}

	if !model.ExperimentTransitions[e.State][model.DeletingState] {
		return nil, fmt.Errorf("cannot delete experiment in %s state", e.State)
	}

	e.State = model.DeletingState
	if err := a.m.db.TrySaveExperimentState(e); err != nil {
		return nil, errors.Wrapf(err, "transitioning to %s", e.State)
	}
	go func() {
		if err := a.deleteExperiment(e, &curUser); err != nil {
			logrus.WithError(err).Errorf("deleting experiment %d", e.ID)
			e.State = model.DeleteFailedState
			if err := a.m.db.SaveExperimentState(e); err != nil {
				logrus.WithError(err).Errorf("transitioning experiment %d to %s", e.ID, e.State)
			}
		} else {
			logrus.Infof("experiment %d deleted successfully", e.ID)
		}
	}()

	return &apiv1.DeleteExperimentResponse{}, nil
}

func (a *apiServer) deleteExperiment(exp *model.Experiment, userModel *model.User) error {
	conf, err := a.m.db.LegacyExperimentConfigByID(exp.ID)
	if err != nil {
		return fmt.Errorf("failed to read config for experiment: %w", err)
	}

	agentUserGroup, err := user.GetAgentUserGroup(*exp.OwnerID, exp)
	if err != nil {
		return err
	}

	taskSpec := *a.m.taskSpec
	checkpoints, err := a.m.db.ExperimentCheckpointsToGCRaw(
		exp.ID,
		0,
		0,
		0,
	)
	if err != nil {
		return err
	}
	addr := actor.Addr(fmt.Sprintf("delete-checkpoint-gc-%s", uuid.New().String()))
	jobSubmissionTime := exp.StartTime
	taskID := model.NewTaskID()
	ckptGCTask := newCheckpointGCTask(
		a.m.rm, a.m.db, a.m.taskLogger, taskID, exp.JobID, jobSubmissionTime, taskSpec, exp.ID,
		conf, checkpoints, true, agentUserGroup, userModel, nil,
	)
	if gcErr := a.m.system.MustActorOf(addr, ckptGCTask).AwaitTermination(); gcErr != nil {
		return errors.Wrapf(gcErr, "failed to gc checkpoints for experiment")
	}

	resp, err := a.m.rm.DeleteJob(a.m.system, sproto.DeleteJob{
		JobID: exp.JobID,
	})
	if err != nil {
		return fmt.Errorf("requesting cleanup of resource mananger resources: %w", err)
	}
	if err = <-resp.Err; err != nil {
		return fmt.Errorf("cleaning up resource mananger resources: %w", err)
	}

	trialIDs, taskIDs, err := a.m.db.ExperimentTrialAndTaskIDs(exp.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to gather trial IDs for experiment")
	}

	if err = a.m.trialLogBackend.DeleteTrialLogs(trialIDs); err != nil {
		return errors.Wrapf(err, "failed to delete trial logs from backend")
	}

	if err = a.m.taskLogBackend.DeleteTaskLogs(taskIDs); err != nil {
		return errors.Wrapf(err, "failed to delete trial logs from backend (task logs)")
	}

	if err = a.m.db.DeleteExperiment(exp.ID); err != nil {
		return errors.Wrapf(err, "deleting experiment from database")
	}
	return nil
}

func protoStateDBCaseString(
	enumToValue map[string]int32, colName, serializedName, trimFromPrefix string,
) string {
	query := fmt.Sprintf("CASE %s::text ", colName)
	for enum, v := range enumToValue {
		query += fmt.Sprintf("WHEN '%s' THEN %d ", strings.TrimPrefix(enum, trimFromPrefix), v)
	}
	return query + fmt.Sprintf("END AS %s", serializedName)
}

func (a *apiServer) GetExperiments(
	ctx context.Context, req *apiv1.GetExperimentsRequest,
) (*apiv1.GetExperimentsResponse, error) {
	resp := &apiv1.GetExperimentsResponse{Experiments: []*experimentv1.Experiment{}}
	query := db.Bun().NewSelect().
		Model(&resp.Experiments).
		ModelTableExpr("experiments as e").
		Column("e.id").
		ColumnExpr("e.config->>'description' AS description").
		ColumnExpr("e.config->>'labels' AS labels").
		ColumnExpr("proto_time(e.start_time) AS start_time").
		ColumnExpr("proto_time(e.end_time) AS end_time").
		ColumnExpr(protoStateDBCaseString(experimentv1.State_value, "e.state", "state", "STATE_")).
		Column("e.archived").
		ColumnExpr(
			"(SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) AS num_trials").
		// Intentionally not sending trial_ids due to performance.
		ColumnExpr("COALESCE(u.display_name, u.username) as display_name").
		ColumnExpr("e.owner_id as user_id").
		Column("u.username").
		ColumnExpr("e.config->'resources'->>'resource_pool' AS resource_pool").
		ColumnExpr("e.config->'searcher'->>'name' AS searcher_type").
		ColumnExpr("e.config->>'name' as NAME").
		ColumnExpr(
			"CASE WHEN NULLIF(e.notes, '') IS NULL THEN NULL ELSE 'omitted' END AS notes").
		Column("e.job_id").
		ColumnExpr("CASE WHEN e.parent_id IS NULL THEN NULL ELSE " +
			"json_build_object('value', e.parent_id) END AS forked_from").
		ColumnExpr("CASE WHEN e.progress IS NULL THEN NULL ELSE " +
			"json_build_object('value', e.progress) END AS progress").
		ColumnExpr("p.name AS project_name").
		Column("e.project_id").
		ColumnExpr("w.id AS workspace_id").
		ColumnExpr("w.name AS workspace_name").
		ColumnExpr("(w.archived OR p.archived) AS parent_archived").
		ColumnExpr("p.user_id AS project_owner_id").
		Column("e.config").
		ColumnExpr("e.group_id").
		ColumnExpr("g.name AS group_name").
		Join("JOIN users u ON e.owner_id = u.id").
		Join("JOIN projects p ON e.project_id = p.id").
		Join("JOIN workspaces w ON p.workspace_id = w.id").
		Join("LEFT JOIN experiment_groups g ON e.group_id = g.id AND e.project_id = g.project_id")

	// Construct the ordering expression.
	orderColMap := map[apiv1.GetExperimentsRequest_SortBy]string{
		apiv1.GetExperimentsRequest_SORT_BY_UNSPECIFIED:   "id",
		apiv1.GetExperimentsRequest_SORT_BY_ID:            "id",
		apiv1.GetExperimentsRequest_SORT_BY_DESCRIPTION:   "description",
		apiv1.GetExperimentsRequest_SORT_BY_NAME:          "name",
		apiv1.GetExperimentsRequest_SORT_BY_START_TIME:    "e.start_time",
		apiv1.GetExperimentsRequest_SORT_BY_END_TIME:      "e.end_time",
		apiv1.GetExperimentsRequest_SORT_BY_STATE:         "e.state",
		apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS:    "num_trials",
		apiv1.GetExperimentsRequest_SORT_BY_PROGRESS:      "COALESCE(progress, 0)",
		apiv1.GetExperimentsRequest_SORT_BY_USER:          "display_name",
		apiv1.GetExperimentsRequest_SORT_BY_FORKED_FROM:   "e.parent_id",
		apiv1.GetExperimentsRequest_SORT_BY_RESOURCE_POOL: "resource_pool",
		apiv1.GetExperimentsRequest_SORT_BY_PROJECT_ID:    "project_id",
		apiv1.GetExperimentsRequest_SORT_BY_GROUP:         "group_name",
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
	query = query.OrderExpr(orderExpr)

	// Filtering
	if req.Description != "" {
		query = query.Where("e.config->>'description' ILIKE ('%%' || ? || '%%')", req.Description)
	}
	if req.Name != "" {
		query = query.Where("e.config->>'name' ILIKE ('%%' || ? || '%%')", req.Name)
	}
	if len(req.Labels) > 0 {
		// In the event labels were removed, if all were removed we insert null,
		// which previously broke this query.
		query = query.Where(`string_to_array(?, ',') <@ ARRAY(SELECT jsonb_array_elements_text(
				CASE WHEN e.config->'labels'::text = 'null'
				THEN NULL
				ELSE e.config->'labels' END
			))`, strings.Join(req.Labels, ",")) // Trying bun.In doesn't work.
	}
	if len(req.Groups) > 0 {
		query = query.Where("g.id IN (?)", bun.In(req.Groups))
	}
	if req.Archived != nil {
		query = query.Where("e.archived = ?", req.Archived.Value)
	}
	if len(req.States) > 0 {
		var allStates []string
		for _, state := range req.States {
			allStates = append(allStates, strings.TrimPrefix(state.String(), "STATE_"))
		}
		query = query.Where("e.state IN (?)", bun.In(allStates))
	}
	if len(req.Users) > 0 {
		query = query.Where("u.username IN (?)", bun.In(req.Users))
	}
	if len(req.UserIds) > 0 {
		query = query.Where("e.owner_id IN (?)", bun.In(req.UserIds))
	}

	if req.ExperimentIdFilter != nil {
		var err error
		query, err = db.ApplyInt32FieldFilter(query, bun.Ident("e.id"), req.ExperimentIdFilter)
		if err != nil {
			return nil, err
		}
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}
	var proj *projectv1.Project
	if req.ProjectId != 0 {
		proj, err = a.GetProjectByID(ctx, req.ProjectId, *curUser)
		if err != nil {
			return nil, err
		}

		query = query.Where("e.project_id = ?", req.ProjectId)
	}
	if query, err = expauth.AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, proj, query); err != nil {
		return nil, err
	}

	resp.Pagination, err = runPagedBunExperimentsQuery(ctx, query, int(req.Offset), int(req.Limit))
	if err != nil {
		return nil, err
	}

	if err = a.enrichExperimentState(resp.Experiments...); err != nil {
		return nil, err
	}

	return resp, nil
}

func runPagedBunExperimentsQuery(
	ctx context.Context, query *bun.SelectQuery, offset, limit int,
) (*apiv1.Pagination, error) {
	// Count number of items without any limits or offsets.
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Calculate end and start indexes.
	startIndex := offset
	if offset > total || offset < -total {
		startIndex = total
	} else if offset < 0 {
		startIndex = total + offset
	}

	endIndex := startIndex + limit
	switch {
	case limit == -2:
		endIndex = startIndex
	case limit == -1:
		endIndex = total
	case limit == 0:
		endIndex = 100 + startIndex
		if total < endIndex {
			endIndex = total
		}
	case startIndex+limit > total:
		endIndex = total
	}

	// Add start and end index to query.
	query.Offset(startIndex)
	query.Limit(endIndex - startIndex)

	// Bun bug treating limit=0 as no limit when it
	// should be the exact opposite of no records returned.
	if endIndex-startIndex != 0 {
		if err = query.Scan(ctx); err != nil {
			return nil, err
		}
	}

	return &apiv1.Pagination{
		Offset:     int32(offset),
		Limit:      int32(limit),
		Total:      int32(total),
		StartIndex: int32(startIndex),
		EndIndex:   int32(endIndex),
	}, nil
}

func (a *apiServer) GetExperimentLabels(ctx context.Context,
	req *apiv1.GetExperimentLabelsRequest,
) (*apiv1.GetExperimentLabelsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	resp := &apiv1.GetExperimentLabelsResponse{}
	var labels [][]string
	query := db.Bun().NewSelect().
		Table("experiments").
		Model(&labels).
		ColumnExpr("config->'labels' AS labels").
		Distinct()

	var proj *projectv1.Project
	if req.ProjectId != 0 {
		proj, err = a.GetProjectByID(ctx, req.ProjectId, *curUser)
		if err != nil {
			return nil, err
		}

		query = query.Where("project_id = ?", req.ProjectId)
	}

	if query, err = expauth.AuthZProvider.Get().
		FilterExperimentLabelsQuery(ctx, *curUser, proj, query); err != nil {
		return nil, err
	}

	if err = query.Scan(ctx); err != nil {
		return nil, err
	}

	// Sort labels by usage.
	labelUsage := make(map[string]int)
	for _, labelArr := range labels {
		for _, l := range labelArr {
			labelUsage[l]++
		}
	}

	resp.Labels = make([]string, len(labelUsage))
	i := 0
	for label := range labelUsage {
		resp.Labels[i] = label
		i++
	}
	sort.Slice(resp.Labels, func(i, j int) bool {
		return labelUsage[resp.Labels[i]] > labelUsage[resp.Labels[j]]
	})
	return resp, nil
}

func (a *apiServer) GetExperimentValidationHistory(
	ctx context.Context, req *apiv1.GetExperimentValidationHistoryRequest,
) (*apiv1.GetExperimentValidationHistoryResponse, error) {
	if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId), false,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	var resp apiv1.GetExperimentValidationHistoryResponse
	switch err := a.m.db.QueryProto("proto_experiment_validation_history", &resp, req.ExperimentId); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "experiment not found: %d", req.ExperimentId)
	case err != nil:
		return nil, errors.Wrapf(err,
			"error fetching validation history for experiment from database: %d", req.ExperimentId)
	}
	return &resp, nil
}

func (a *apiServer) PreviewHPSearch(
	ctx context.Context, req *apiv1.PreviewHPSearchRequest,
) (*apiv1.PreviewHPSearchResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = expauth.AuthZProvider.Get().CanPreviewHPSearch(ctx, *curUser); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	bytes, err := protojson.Marshal(req.Config)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error parsing experiment config: %s", err)
	}

	// Parse the provided experiment config.
	config, err := expconf.ParseAnyExperimentConfigYAML(bytes)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "invalid experiment configuration: %s", err,
		)
	}

	// Get the useful subconfigs for preview search.
	if config.RawSearcher == nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "invalid experiment configuration; missing searcher",
		)
	}
	sc := *config.RawSearcher
	hc := config.RawHyperparameters

	// Apply any json-schema-defined defaults.
	sc = schemas.WithDefaults(sc).(expconf.SearcherConfig)
	hc = schemas.WithDefaults(hc).(expconf.Hyperparameters)

	// Make sure the searcher config has all eventuallyRequired fields.
	if err = schemas.IsComplete(sc); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid searcher configuration: %s", err)
	}
	if err = schemas.IsComplete(hc); err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "invalid hyperparameters configuration: %s", err,
		)
	}

	// Disallow EOL searchers.
	if err = sc.AssertCurrent(); err != nil {
		return nil, errors.Wrap(err, "invalid experiment configuration")
	}

	sm := searcher.NewSearchMethod(sc)
	s := searcher.NewSearcher(req.Seed, sm, hc)
	sim, err := searcher.Simulate(s, nil, searcher.RandomValidation, true, sc.Metric())
	if err != nil {
		return nil, err
	}
	protoSim := &experimentv1.ExperimentSimulation{Seed: req.Seed}
	indexes := make(map[string]int, len(sim.Results))
	toProto := func(op searcher.ValidateAfter) ([]*experimentv1.RunnableOperation, error) {
		return []*experimentv1.RunnableOperation{
			{
				Type:   experimentv1.RunnableType_RUNNABLE_TYPE_TRAIN,
				Length: op.Length,
			},
			{
				Type: experimentv1.RunnableType_RUNNABLE_TYPE_VALIDATE,
			},
		}, nil
	}
	for _, result := range sim.Results {
		var operations []*experimentv1.RunnableOperation
		for _, msg := range result {
			ops, err := toProto(msg)
			if err != nil {
				return nil, errors.Wrapf(err, "error converting msg in simultion result %s", msg)
			}
			operations = append(operations, ops...)
		}
		hash := fmt.Sprint(operations)
		if i, ok := indexes[hash]; ok {
			protoSim.Trials[i].Occurrences++
		} else {
			protoSim.Trials = append(protoSim.Trials,
				&experimentv1.TrialSimulation{Operations: operations, Occurrences: 1})
			indexes[hash] = len(protoSim.Trials) - 1
		}
	}
	return &apiv1.PreviewHPSearchResponse{Simulation: protoSim}, nil
}

func (a *apiServer) ActivateExperiment(
	ctx context.Context, req *apiv1.ActivateExperimentRequest,
) (resp *apiv1.ActivateExperimentResponse, err error) {
	if _, _, err = a.getExperimentAndCheckCanDoActions(ctx, int(req.Id), false,
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	addr := experimentsAddr.Child(req.Id)
	switch err = a.ask(addr, req, &resp); {
	case status.Code(err) == codes.NotFound:
		return nil, status.Error(codes.FailedPrecondition, "experiment in terminal state")
	case err != nil:
		return nil, status.Errorf(codes.Internal, "failed passing request to experiment actor: %s", err)
	default:
		return resp, nil
	}
}

func (a *apiServer) PauseExperiment(
	ctx context.Context, req *apiv1.PauseExperimentRequest,
) (resp *apiv1.PauseExperimentResponse, err error) {
	if _, _, err = a.getExperimentAndCheckCanDoActions(ctx, int(req.Id), false,
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	addr := experimentsAddr.Child(req.Id)
	switch err = a.ask(addr, req, &resp); {
	case status.Code(err) == codes.NotFound:
		return nil, status.Error(codes.FailedPrecondition, "experiment in terminal state")
	case err != nil:
		return nil, status.Errorf(codes.Internal, "failed passing request to experiment actor: %s", err)
	default:
		return resp, nil
	}
}

func (a *apiServer) CancelExperiment(
	ctx context.Context, req *apiv1.CancelExperimentRequest,
) (resp *apiv1.CancelExperimentResponse, err error) {
	if _, _, err = a.getExperimentAndCheckCanDoActions(ctx, int(req.Id), false,
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	addr := experimentsAddr.Child(req.Id)
	err = a.ask(addr, req, &resp)
	if status.Code(err) == codes.NotFound {
		return &apiv1.CancelExperimentResponse{}, nil
	}
	return resp, err
}

func (a *apiServer) KillExperiment(
	ctx context.Context, req *apiv1.KillExperimentRequest,
) (resp *apiv1.KillExperimentResponse, err error) {
	if _, _, err = a.getExperimentAndCheckCanDoActions(ctx, int(req.Id), false,
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	addr := experimentsAddr.Child(req.Id)
	err = a.ask(addr, req, &resp)
	if status.Code(err) == codes.NotFound {
		return &apiv1.KillExperimentResponse{}, nil
	}
	return resp, err
}

func (a *apiServer) ArchiveExperiment(
	ctx context.Context, req *apiv1.ArchiveExperimentRequest,
) (*apiv1.ArchiveExperimentResponse, error) {
	id := int(req.Id)
	dbExp, _, err := a.getExperimentAndCheckCanDoActions(ctx, id, false,
		expauth.AuthZProvider.Get().CanEditExperimentsMetadata)
	if err != nil {
		return nil, err
	}
	if _, ok := model.TerminalStates[dbExp.State]; !ok {
		return nil, errors.Errorf("cannot archive experiment %v in non terminate state %v",
			id, dbExp.State)
	}

	if dbExp.Archived {
		return &apiv1.ArchiveExperimentResponse{}, nil
	}
	dbExp.Archived = true
	err = a.m.db.SaveExperimentArchiveStatus(dbExp)
	switch err {
	case nil:
		return &apiv1.ArchiveExperimentResponse{}, nil
	default:
		return nil, errors.Wrapf(err, "failed to archive experiment %d",
			req.Id)
	}
}

func (a *apiServer) UnarchiveExperiment(
	ctx context.Context, req *apiv1.UnarchiveExperimentRequest,
) (*apiv1.UnarchiveExperimentResponse, error) {
	id := int(req.Id)
	dbExp, _, err := a.getExperimentAndCheckCanDoActions(ctx, id, false,
		expauth.AuthZProvider.Get().CanEditExperimentsMetadata)
	if err != nil {
		return nil, err
	}

	if _, ok := model.TerminalStates[dbExp.State]; !ok {
		return nil, errors.Errorf("cannot unarchive experiment %v in non terminate state %v",
			id, dbExp.State)
	}

	if !dbExp.Archived {
		return &apiv1.UnarchiveExperimentResponse{}, nil
	}
	dbExp.Archived = false
	err = a.m.db.SaveExperimentArchiveStatus(dbExp)
	switch err {
	case nil:
		return &apiv1.UnarchiveExperimentResponse{}, nil
	default:
		return nil, errors.Wrapf(err, "failed to archive experiment %d",
			req.Id)
	}
}

func maskContains(mask []string, v string) bool {
	if len(mask) == 0 {
		return true
	}
	for _, s := range mask {
		if v == s {
			return true
		}
	}
	return false
}

func (a *apiServer) PatchExperiment(
	ctx context.Context, req *apiv1.PatchExperimentRequest,
) (*apiv1.PatchExperimentResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	exp, err := a.getExperiment(ctx, *curUser, int(req.Experiment.Id))
	if err != nil {
		return nil, err
	}
	modelExp, err := model.ExperimentFromProto(exp)
	if err != nil {
		return nil, err
	}

	if err = expauth.AuthZProvider.Get().CanEditExperimentsMetadata(
		ctx, *curUser, modelExp); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	var mask []string

	if req.UpdateMask != nil {
		req.UpdateMask.Normalize()
		mask = req.UpdateMask.GetPaths()
	}

	if len(mask) > 0 && !req.UpdateMask.IsValid(req.Experiment) {
		return nil, status.Errorf(codes.InvalidArgument,
			"update mask must correspond to experiment type.")
	}

	madeChanges := false
	if req.Experiment.Name != nil && exp.Name != req.Experiment.Name.Value &&
		maskContains(mask, "name") {
		madeChanges = true
		if len(strings.TrimSpace(req.Experiment.Name.Value)) == 0 {
			return nil, status.Errorf(codes.InvalidArgument,
				"`name` must not be an empty or whitespace string.")
		}
		exp.Name = req.Experiment.Name.Value
	}

	if req.Experiment.Notes != nil && exp.Notes != req.Experiment.Notes.Value &&
		maskContains(mask, "notes") {
		madeChanges = true
		exp.Notes = req.Experiment.Notes.Value
	}

	if req.Experiment.Description != nil && exp.Description != req.Experiment.Description.Value &&
		maskContains(mask, "description") {
		madeChanges = true
		exp.Description = req.Experiment.Description.Value
	}

	if req.Experiment.Labels != nil && maskContains(mask, "labels") {
		var reqLabelList []string
		for _, el := range req.Experiment.Labels.Values {
			if _, ok := el.GetKind().(*structpb.Value_StringValue); ok {
				reqLabelList = append(reqLabelList, el.GetStringValue())
			}
		}
		reqLabels := strings.Join(reqLabelList, ",")
		if strings.Join(exp.Labels, ",") != reqLabels {
			madeChanges = true
			exp.Labels = reqLabelList
			prom.AssociateExperimentIDLabels(strconv.Itoa(int(req.Experiment.Id)),
				exp.Labels)
		}
	}

	if exp.GroupId != req.Experiment.GroupId && maskContains(mask, "group_id") {
		if req.Experiment.GroupId != nil {
			var destGroup *projectv1.ExperimentGroup
			destGroup, err = a.GetExperimentGroupByID(ctx, *req.Experiment.GroupId)
			if err != nil {
				return nil, err
			}
			madeChanges = true
			exp.GroupId = &destGroup.Id
			exp.GroupName = &destGroup.Name
		} else if len(mask) > 0 {
			madeChanges = true
			exp.GroupId = nil
			exp.GroupName = nil
		}
	}

	if madeChanges {
		type experimentPatch struct {
			Labels      []string `json:"labels"`
			Description string   `json:"description"`
			Name        string   `json:"name"`
		}
		patches := experimentPatch{
			Labels:      exp.Labels,
			Description: exp.Description,
			Name:        exp.Name,
		}
		marshalledPatches, patchErr := json.Marshal(patches)
		if patchErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal experiment patch")
		}

		_, err = a.m.db.RawQuery(
			"patch_experiment", exp.Id, marshalledPatches, exp.Notes, exp.GroupId)
		if err != nil {
			return nil, errors.Wrapf(err, "error updating experiment in database: %d", req.Experiment.Id)
		}
	}

	// include queued / pulling / starting / running state
	if err = a.enrichExperimentState(exp); err != nil {
		return nil, err
	}

	return &apiv1.PatchExperimentResponse{Experiment: exp}, nil
}

func (a *apiServer) GetExperimentCheckpoints(
	ctx context.Context, req *apiv1.GetExperimentCheckpointsRequest,
) (*apiv1.GetExperimentCheckpointsResponse, error) {
	experimentID := int(req.Id)
	useSearcherSortBy := req.SortBy == apiv1.GetExperimentCheckpointsRequest_SORT_BY_SEARCHER_METRIC
	exp, _, err := a.getExperimentAndCheckCanDoActions(ctx, experimentID, useSearcherSortBy,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts)
	if err != nil {
		return nil, err
	}

	// If SORT_BY_SEARCHER_METRIC is specified without an OrderBy
	// default to ordering by "better" checkpoints.
	if useSearcherSortBy && req.OrderBy == apiv1.OrderBy_ORDER_BY_UNSPECIFIED {
		if exp.Config.Searcher().SmallerIsBetter() {
			req.OrderBy = apiv1.OrderBy_ORDER_BY_ASC
		} else {
			req.OrderBy = apiv1.OrderBy_ORDER_BY_DESC
		}
	}

	resp := &apiv1.GetExperimentCheckpointsResponse{}
	resp.Checkpoints = []*checkpointv1.Checkpoint{}
	switch err = a.m.db.QueryProto("get_checkpoints_for_experiment", &resp.Checkpoints, req.Id); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "no checkpoints found for experiment %d", req.Id)
	case err != nil:
		return nil,
			errors.Wrapf(err, "error fetching checkpoints for experiment %d from database", req.Id)
	}

	a.filter(&resp.Checkpoints, func(i int) bool {
		v := resp.Checkpoints[i]

		found := false
		for _, state := range req.States {
			if state == v.State {
				found = true
				break
			}
		}

		if len(req.States) != 0 && !found {
			return false
		}

		return true
	})

	sort.Slice(resp.Checkpoints, func(i, j int) bool {
		ai, aj := resp.Checkpoints[i], resp.Checkpoints[j]
		if useSearcherSortBy {
			if order, done := protoless.CheckpointSearcherMetricNullsLast(ai, aj); done {
				return order
			}
		}

		if req.OrderBy == apiv1.OrderBy_ORDER_BY_DESC {
			aj, ai = ai, aj
		}

		switch req.SortBy {
		case apiv1.GetExperimentCheckpointsRequest_SORT_BY_BATCH_NUMBER:
			return protoless.CheckpointStepsCompletedLess(ai, aj)
		case apiv1.GetExperimentCheckpointsRequest_SORT_BY_UUID:
			return ai.Uuid < aj.Uuid
		case apiv1.GetExperimentCheckpointsRequest_SORT_BY_TRIAL_ID:
			return protoless.CheckpointTrialIDLess(ai, aj)
		case apiv1.GetExperimentCheckpointsRequest_SORT_BY_END_TIME:
			return protoless.CheckpointReportTimeLess(ai, aj)
		case apiv1.GetExperimentCheckpointsRequest_SORT_BY_STATE:
			return ai.State.Number() < aj.State.Number()
		case apiv1.GetExperimentCheckpointsRequest_SORT_BY_SEARCHER_METRIC:
			return protoless.CheckpointSearcherMetricLess(ai, aj)
		case apiv1.GetExperimentCheckpointsRequest_SORT_BY_UNSPECIFIED:
			fallthrough
		default:
			return protoless.CheckpointTrialIDLess(ai, aj)
		}
	})
	return resp, a.paginate(&resp.Pagination, &resp.Checkpoints, req.Offset, req.Limit)
}

func (a *apiServer) CreateExperiment(
	ctx context.Context, req *apiv1.CreateExperimentRequest,
) (*apiv1.CreateExperimentResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	detParams := CreateExperimentParams{
		ConfigBytes:  req.Config,
		ModelDef:     filesToArchive(req.ModelDefinition),
		ValidateOnly: req.ValidateOnly,
	}
	if req.ParentId != 0 {
		detParams.ParentID = ptrs.Ptr(int(req.ParentId))
		// Can't use getExperimentAndCheckDoActions since model.Experiment doesn't have ParentArchived.
		var parentExp *experimentv1.Experiment
		parentExp, err = a.getExperiment(ctx, *user, *detParams.ParentID)
		if err != nil {
			return nil, err
		}
		var modelExp *model.Experiment
		modelExp, err = model.ExperimentFromProto(parentExp)
		if err != nil {
			return nil, err
		}

		if err = expauth.AuthZProvider.Get().
			CanForkFromExperiment(ctx, *user, modelExp); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
		if parentExp.ParentArchived {
			return nil, status.Errorf(codes.Internal,
				"forking an experiment in an archived workspace/project")
		}
	}
	if req.ProjectId > 1 {
		projectID := int(req.ProjectId)
		detParams.ProjectID = &projectID
	}

	dbExp, p, validateOnly, taskSpec, err := a.m.parseCreateExperiment(&detParams, user)
	if err != nil {
		if _, ok := err.(ErrProjectNotFound); ok {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid experiment: %s", err)
	}
	if err = expauth.AuthZProvider.Get().CanCreateExperiment(ctx, *user, p, dbExp); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	if validateOnly {
		return &apiv1.CreateExperimentResponse{}, nil
	}
	// Check user has permission for what they are trying to do
	// before actually saving the experiment.
	if req.Activate {
		if err = expauth.AuthZProvider.Get().CanEditExperiment(ctx, *user, dbExp); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
	}

	e, err := newExperiment(a.m, dbExp, taskSpec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create experiment: %s", err)
	}
	a.m.system.ActorOf(experimentsAddr.Child(e.ID), e)

	if req.Activate {
		_, err = a.ActivateExperiment(ctx, &apiv1.ActivateExperimentRequest{Id: int32(e.ID)})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to activate experiment: %s", err)
		}
	}

	protoExp, err := a.getExperiment(ctx, *user, e.ID)
	if err != nil {
		return nil, err
	}
	return &apiv1.CreateExperimentResponse{
		Experiment: protoExp, Config: protoutils.ToStruct(e.Config),
	}, nil
}

var (
	defaultMetricsStreamPeriod = 30 * time.Second
	recheckAuthPeriod          = 5 * time.Minute
)

func (a *apiServer) MetricNames(req *apiv1.MetricNamesRequest,
	resp apiv1.Determined_MetricNamesServer,
) error {
	experimentID := int(req.ExperimentId)
	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	seenTrain := make(map[string]bool)
	seenValid := make(map[string]bool)
	var tStartTime time.Time
	var vStartTime time.Time

	var timeSinceLastAuth time.Time
	var config expconf.ExperimentConfig
	var searcherMetric string
	for {
		if time.Now().Sub(timeSinceLastAuth) >= recheckAuthPeriod {
			if _, _, err := a.getExperimentAndCheckCanDoActions(resp.Context(), experimentID,
				false, expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return err
			}

			if timeSinceLastAuth == (time.Time{}) { // Initialzation.
				var err error
				config, err = a.m.db.ExperimentConfig(experimentID)
				if err != nil {
					return errors.Wrapf(err,
						"error fetching experiment config from database: %d", experimentID)
				}
				searcherMetric = config.Searcher().Metric()
			}
			timeSinceLastAuth = time.Now()
		}

		var response apiv1.MetricNamesResponse
		response.SearcherMetric = searcherMetric

		newTrain, newValid, tEndTime, vEndTime, err := a.m.db.MetricNames(experimentID,
			tStartTime, vStartTime)
		if err != nil {
			return errors.Wrapf(err,
				"error fetching metric names for experiment: %d", experimentID)
		}
		tStartTime = tEndTime
		vStartTime = vEndTime

		for _, name := range newTrain {
			if seen := seenTrain[name]; !seen {
				response.TrainingMetrics = append(response.TrainingMetrics, name)
				seenTrain[name] = true
			}
		}
		for _, name := range newValid {
			if seen := seenValid[name]; !seen {
				response.ValidationMetrics = append(response.ValidationMetrics, name)
				seenValid[name] = true
			}
		}

		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
		if err = resp.Send(&response); err != nil {
			return err
		}

		state, _, err := a.m.db.GetExperimentStatus(experimentID)
		if err != nil {
			return errors.Wrap(err, "error looking up experiment state")
		}
		if model.TerminalStates[state] {
			return nil
		}

		time.Sleep(period)
		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
	}
}

// DEPRECATED -- do not use.
func (a *apiServer) ExpCompareMetricNames(req *apiv1.ExpCompareMetricNamesRequest,
	resp apiv1.Determined_ExpCompareMetricNamesServer,
) error {
	seenTrain := make(map[string]bool)
	seenValid := make(map[string]bool)
	var tStartTime time.Time
	var vStartTime time.Time
	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}
	if len(req.TrialId) == 0 {
		return status.Errorf(
			codes.InvalidArgument,
			"at least one trial id required",
		)
	}

	var timeSinceLastAuth time.Time
	for {
		if time.Now().Sub(timeSinceLastAuth) >= recheckAuthPeriod {
			for _, trialID := range req.TrialId {
				if err := a.canGetTrialsExperimentAndCheckCanDoAction(resp.Context(), int(trialID),
					expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
					return err
				}
			}
			timeSinceLastAuth = time.Now()
		}

		var response apiv1.ExpCompareMetricNamesResponse

		newTrain, newValid, tEndTime, vEndTime, err := a.m.db.ExpCompareMetricNames(req.TrialId,
			tStartTime, vStartTime)
		if err != nil {
			return err
		}
		tStartTime = tEndTime
		vStartTime = vEndTime

		for _, name := range newTrain {
			if seen := seenTrain[name]; !seen {
				response.TrainingMetrics = append(response.TrainingMetrics, name)
				seenTrain[name] = true
			}
		}
		for _, name := range newValid {
			if seen := seenValid[name]; !seen {
				response.ValidationMetrics = append(response.ValidationMetrics, name)
				seenValid[name] = true
			}
		}

		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
		if err = resp.Send(&response); err != nil {
			return err
		}

		time.Sleep(period)
		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
	}
}

func (a *apiServer) MetricBatches(req *apiv1.MetricBatchesRequest,
	resp apiv1.Determined_MetricBatchesServer,
) error {
	experimentID := int(req.ExperimentId)
	metricName := req.MetricName
	if metricName == "" {
		return status.Error(codes.InvalidArgument, "must specify a metric name")
	}
	metricType := req.MetricType
	if metricType == apiv1.MetricType_METRIC_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "must specify a metric type")
	}
	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	var timeSinceLastAuth time.Time
	seenBatches := make(map[int32]bool)
	var startTime time.Time
	for {
		if time.Now().Sub(timeSinceLastAuth) >= recheckAuthPeriod {
			if _, _, err := a.getExperimentAndCheckCanDoActions(resp.Context(), experimentID, false,
				expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return err
			}
			timeSinceLastAuth = time.Now()
		}

		var response apiv1.MetricBatchesResponse

		var newBatches []int32
		var endTime time.Time
		var err error
		switch metricType {
		case apiv1.MetricType_METRIC_TYPE_TRAINING:
			newBatches, endTime, err = a.m.db.TrainingMetricBatches(experimentID, metricName,
				startTime)
		case apiv1.MetricType_METRIC_TYPE_VALIDATION:
			newBatches, endTime, err = a.m.db.ValidationMetricBatches(experimentID, metricName,
				startTime)
		default:
			panic("Invalid metric type")
		}
		if err != nil {
			return errors.Wrapf(err, "error fetching batches recorded for metric")
		}
		startTime = endTime

		for _, batch := range newBatches {
			if seen := seenBatches[batch]; !seen {
				response.Batches = append(response.Batches, batch)
				seenBatches[batch] = true
			}
		}

		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
		if err = resp.Send(&response); err != nil {
			return errors.Wrapf(err, "error sending batches recorded for metric")
		}

		state, _, err := a.m.db.GetExperimentStatus(experimentID)
		if err != nil {
			return errors.Wrap(err, "error looking up experiment state")
		}
		if model.TerminalStates[state] {
			return nil
		}

		time.Sleep(period)
		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
	}
}

func (a *apiServer) TrialsSnapshot(req *apiv1.TrialsSnapshotRequest,
	resp apiv1.Determined_TrialsSnapshotServer,
) error {
	experimentID := int(req.ExperimentId)
	metricName := req.MetricName
	if metricName == "" {
		return status.Error(codes.InvalidArgument, "must specify a metric name")
	}
	metricType := req.MetricType
	if metricType == apiv1.MetricType_METRIC_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "must specify a metric type")
	}
	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	batchesProcessed := int(req.BatchesProcessed)
	batchesMargin := int(req.BatchesMargin)
	if batchesMargin > 100 {
		return status.Error(codes.InvalidArgument, "margin must be <= 100")
	}
	minBatches := batchesProcessed - batchesMargin
	if minBatches < 0 {
		minBatches = 0
	}
	maxBatches := batchesProcessed + batchesMargin
	if maxBatches < 0 {
		maxBatches = math.MaxInt32
	}

	var timeSinceLastAuth time.Time
	var startTime time.Time
	for {
		if time.Now().Sub(timeSinceLastAuth) >= recheckAuthPeriod {
			if _, _, err := a.getExperimentAndCheckCanDoActions(resp.Context(), experimentID, false,
				expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return err
			}
			timeSinceLastAuth = time.Now()
		}

		var response apiv1.TrialsSnapshotResponse
		var newTrials []*apiv1.TrialsSnapshotResponse_Trial
		var endTime time.Time
		var err error
		switch metricType {
		case apiv1.MetricType_METRIC_TYPE_TRAINING:
			newTrials, endTime, err = a.m.db.TrainingTrialsSnapshot(experimentID,
				minBatches, maxBatches, metricName, startTime)
		case apiv1.MetricType_METRIC_TYPE_VALIDATION:
			newTrials, endTime, err = a.m.db.ValidationTrialsSnapshot(experimentID,
				minBatches, maxBatches, metricName, startTime)
		default:
			panic("Invalid metric type")
		}
		if err != nil {
			return errors.Wrapf(err,
				"error fetching snapshots of metrics for %s metric %s in experiment %d at %d batches",
				metricType, metricName, experimentID, batchesProcessed)
		}
		startTime = endTime

		response.Trials = newTrials

		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
		if err = resp.Send(&response); err != nil {
			return errors.Wrapf(err, "error sending batches recorded for metrics")
		}

		state, _, err := a.m.db.GetExperimentStatus(experimentID)
		if err != nil {
			return errors.Wrap(err, "error looking up experiment state")
		}
		if model.TerminalStates[state] {
			return nil
		}

		time.Sleep(period)
		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
	}
}

func (a *apiServer) topTrials(experimentID int, maxTrials int, s expconf.SearcherConfig) (
	trials []int32, err error,
) {
	type Ranking int
	const (
		ByMetricOfInterest Ranking = 1
		ByTrainingLength   Ranking = 2
	)
	var ranking Ranking

	switch s.GetUnionMember().(type) {
	case expconf.RandomConfig:
		ranking = ByMetricOfInterest
	case expconf.GridConfig:
		ranking = ByMetricOfInterest
	case expconf.CustomConfig:
		ranking = ByMetricOfInterest
	case expconf.AsyncHalvingConfig:
		ranking = ByTrainingLength
	case expconf.AdaptiveASHAConfig:
		ranking = ByTrainingLength
	case expconf.SingleConfig:
		return nil, errors.New("single-trial experiments are not supported for trial sampling")
	// EOL searcher configs:
	case expconf.AdaptiveConfig:
		ranking = ByTrainingLength
	case expconf.AdaptiveSimpleConfig:
		ranking = ByTrainingLength
	case expconf.SyncHalvingConfig:
		ranking = ByTrainingLength
	default:
		return nil, errors.New("unable to detect a searcher algorithm for trial sampling")
	}
	switch ranking {
	case ByMetricOfInterest:
		return a.m.db.TopTrialsByMetric(experimentID, maxTrials, s.Metric(), s.SmallerIsBetter())
	case ByTrainingLength:
		return a.m.db.TopTrialsByTrainingLength(experimentID, maxTrials, s.Metric(), s.SmallerIsBetter())
	default:
		panic("Invalid state in trial sampling")
	}
}

func (a *apiServer) fetchTrialSample(trialID int32, metricName string, metricType apiv1.MetricType,
	maxDatapoints int, startBatches int, endBatches int, currentTrials map[int32]bool,
	trialCursors map[int32]time.Time,
) (*apiv1.TrialsSampleResponse_Trial, error) {
	var metricSeries []lttb.Point
	var endTime time.Time
	var zeroTime time.Time
	var err error
	var trial apiv1.TrialsSampleResponse_Trial
	trial.TrialId = trialID

	if _, current := currentTrials[trialID]; !current {
		var trialConfig *model.Trial
		trialConfig, err = a.m.db.TrialByID(int(trialID))
		if err != nil {
			return nil, errors.Wrapf(err, "error fetching trial metadata")
		}
		trial.Hparams = protoutils.ToStruct(trialConfig.HParams)
	}

	startTime, seenBefore := trialCursors[trialID]
	if !seenBefore {
		startTime = zeroTime
	}
	switch metricType {
	case apiv1.MetricType_METRIC_TYPE_TRAINING:
		metricSeries, endTime, err = a.m.db.TrainingMetricsSeries(trialID, startTime,
			metricName, startBatches, endBatches)
	case apiv1.MetricType_METRIC_TYPE_VALIDATION:
		metricSeries, endTime, err = a.m.db.ValidationMetricsSeries(trialID, startTime,
			metricName, startBatches, endBatches)
	default:
		panic("Invalid metric type")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching time series of metrics")
	}
	if len(metricSeries) > 0 {
		// if we get empty results, the endTime is incorrectly zero
		trialCursors[trialID] = endTime
	}
	if !seenBefore {
		metricSeries = lttb.Downsample(metricSeries, maxDatapoints, false)
	}

	for _, in := range metricSeries {
		out := apiv1.DataPoint{
			Batches: int32(in.X),
			Value:   in.Y,
		}
		trial.Data = append(trial.Data, &out)
	}
	return &trial, nil
}

// DEPRECATED -- do not use.
func (a *apiServer) expCompareFetchTrialSample(trialID int32, metricName string,
	metricType apiv1.MetricType, maxDatapoints int, startBatches int, endBatches int,
	currentTrials map[int32]bool,
	trialCursors map[int32]time.Time,
) (*apiv1.ExpCompareTrialsSampleResponse_ExpTrial, error) {
	var metricSeries []lttb.Point
	var endTime time.Time
	var zeroTime time.Time
	var err error
	var trial apiv1.ExpCompareTrialsSampleResponse_ExpTrial
	trial.TrialId = trialID

	if _, current := currentTrials[trialID]; !current {
		var trialConfig *model.Trial
		trialConfig, err = a.m.db.TrialByID(int(trialID))
		if err != nil {
			return nil, errors.Wrapf(err, "error fetching trial metadata")
		}
		trial.Hparams = protoutils.ToStruct(trialConfig.HParams)
		trial.ExperimentId = int32(trialConfig.ExperimentID)
	}

	startTime, seenBefore := trialCursors[trialID]
	if !seenBefore {
		startTime = zeroTime
	}
	switch metricType {
	case apiv1.MetricType_METRIC_TYPE_TRAINING:
		metricSeries, endTime, err = a.m.db.TrainingMetricsSeries(trialID, startTime,
			metricName, startBatches, endBatches)
	case apiv1.MetricType_METRIC_TYPE_VALIDATION:
		metricSeries, endTime, err = a.m.db.ValidationMetricsSeries(trialID, startTime,
			metricName, startBatches, endBatches)
	default:
		panic("Invalid metric type")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching time series of metrics")
	}
	if len(metricSeries) > 0 {
		// if we get empty results, the endTime is incorrectly zero
		trialCursors[trialID] = endTime
	}
	if !seenBefore {
		metricSeries = lttb.Downsample(metricSeries, maxDatapoints, false)
	}

	for _, in := range metricSeries {
		out := apiv1.DataPoint{
			Batches: int32(in.X),
			Value:   in.Y,
		}
		trial.Data = append(trial.Data, &out)
	}
	return &trial, nil
}

func (a *apiServer) TrialsSample(req *apiv1.TrialsSampleRequest,
	resp apiv1.Determined_TrialsSampleServer,
) error {
	experimentID := int(req.ExperimentId)
	maxTrials := int(req.MaxTrials)
	if maxTrials == 0 {
		maxTrials = 25
	}
	maxDatapoints := int(req.MaxDatapoints)
	if maxDatapoints == 0 {
		maxDatapoints = 1000
	}
	startBatches := int(req.StartBatches)
	endBatches := int(req.EndBatches)
	if endBatches <= 0 {
		endBatches = math.MaxInt32
	}
	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	metricName := req.MetricName
	metricType := req.MetricType
	if metricType == apiv1.MetricType_METRIC_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "must specify a metric type")
	}
	if metricName == "" {
		return status.Error(codes.InvalidArgument, "must specify a metric name")
	}

	var timeSinceLastAuth time.Time
	var config expconf.ExperimentConfig
	var searcherConfig expconf.SearcherConfig
	trialCursors := make(map[int32]time.Time)
	currentTrials := make(map[int32]bool)
	for {
		if time.Now().Sub(timeSinceLastAuth) >= recheckAuthPeriod {
			if _, _, err := a.getExperimentAndCheckCanDoActions(resp.Context(), experimentID, false,
				expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return err
			}

			if timeSinceLastAuth == (time.Time{}) { // Initialzation.
				var err error
				config, err = a.m.db.ExperimentConfig(experimentID)
				if err != nil {
					return errors.Wrapf(err, "error fetching experiment config from database")
				}
				searcherConfig = config.Searcher()
			}
			timeSinceLastAuth = time.Now()
		}

		var response apiv1.TrialsSampleResponse
		var promotedTrials []int32
		var demotedTrials []int32
		var trials []*apiv1.TrialsSampleResponse_Trial

		seenThisRound := make(map[int32]bool)

		trialIDs, err := a.topTrials(experimentID, maxTrials, searcherConfig)
		if err != nil {
			return errors.Wrapf(err, "error determining top trials")
		}
		for _, trialID := range trialIDs {
			var trial *apiv1.TrialsSampleResponse_Trial
			trial, err = a.fetchTrialSample(trialID, metricName, metricType, maxDatapoints,
				startBatches, endBatches, currentTrials, trialCursors)
			if err != nil {
				return err
			}

			if _, current := currentTrials[trialID]; !current {
				promotedTrials = append(promotedTrials, trialID)
				currentTrials[trialID] = true
			}
			seenThisRound[trialID] = true

			trials = append(trials, trial)
		}
		for oldTrial := range currentTrials {
			if !seenThisRound[oldTrial] {
				demotedTrials = append(demotedTrials, oldTrial)
				delete(trialCursors, oldTrial)
			}
		}
		// Deletes from currentTrials have to happen when not looping over currentTrials
		for _, oldTrial := range demotedTrials {
			delete(currentTrials, oldTrial)
		}

		response.Trials = trials
		response.PromotedTrials = promotedTrials
		response.DemotedTrials = demotedTrials

		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
		if err = resp.Send(&response); err != nil {
			return errors.Wrap(err, "error sending sample of trial metric streams")
		}

		state, _, err := a.m.db.GetExperimentStatus(experimentID)
		if err != nil {
			return errors.Wrap(err, "error looking up experiment state")
		}
		if model.TerminalStates[state] {
			return nil
		}

		time.Sleep(period)
		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
	}
}

func (a *apiServer) ExpCompareTrialsSample(req *apiv1.ExpCompareTrialsSampleRequest,
	resp apiv1.Determined_ExpCompareTrialsSampleServer,
) error {
	experimentIDs := req.ExperimentIds
	maxTrials := int(req.MaxTrials)
	if maxTrials == 0 {
		maxTrials = 25
	}
	maxDatapoints := int(req.MaxDatapoints)
	if maxDatapoints == 0 {
		maxDatapoints = 1000
	}
	startBatches := int(req.StartBatches)
	endBatches := int(req.EndBatches)
	if endBatches <= 0 {
		endBatches = math.MaxInt32
	}
	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	metricName := req.MetricName
	metricType := req.MetricType
	if metricType == apiv1.MetricType_METRIC_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "must specify a metric type")
	}
	if metricName == "" {
		return status.Error(codes.InvalidArgument, "must specify a metric name")
	}

	var timeSinceLastAuth time.Time
	trialCursors := make(map[int32]time.Time)
	currentTrials := make(map[int32]bool)
	for {
		if time.Now().Sub(timeSinceLastAuth) >= recheckAuthPeriod {
			for _, expID := range experimentIDs {
				if _, _, err := a.getExperimentAndCheckCanDoActions(resp.Context(), int(expID), false,
					expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
					return err
				}
			}
			timeSinceLastAuth = time.Now()
		}

		var response apiv1.ExpCompareTrialsSampleResponse
		var promotedTrials []int32
		var demotedTrials []int32
		var trials []*apiv1.ExpCompareTrialsSampleResponse_ExpTrial

		seenThisRound := make(map[int32]bool)

		r := regexp.MustCompile("(?i)(error|loss|mse|mae|mse|deviation|false)")
		smallerIsBetter := r.MatchString(metricName)

		trialIDs, err := a.m.db.ExpCompareTopTrialsByMetric(experimentIDs,
			maxTrials,
			metricName,
			smallerIsBetter)
		if err != nil {
			return err
		}

		for _, trialID := range trialIDs {
			var trial *apiv1.ExpCompareTrialsSampleResponse_ExpTrial
			trial, err = a.expCompareFetchTrialSample(trialID, metricName, metricType, maxDatapoints,
				startBatches, endBatches, currentTrials, trialCursors)
			if err != nil {
				return err
			}

			if _, current := currentTrials[trialID]; !current {
				promotedTrials = append(promotedTrials, trialID)
				currentTrials[trialID] = true
			}
			seenThisRound[trialID] = true

			trials = append(trials, trial)
		}
		for oldTrial := range currentTrials {
			if !seenThisRound[oldTrial] {
				demotedTrials = append(demotedTrials, oldTrial)
				delete(trialCursors, oldTrial)
			}
		}
		// Deletes from currentTrials have to happen when not looping over currentTrials
		for _, oldTrial := range demotedTrials {
			delete(currentTrials, oldTrial)
		}

		response.Trials = trials
		response.PromotedTrials = promotedTrials
		response.DemotedTrials = demotedTrials

		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
		if err = resp.Send(&response); err != nil {
			return errors.Wrap(err, "error sending sample of trial metric streams")
		}

		time.Sleep(period)
		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
	}
}

func (a *apiServer) ComputeHPImportance(ctx context.Context,
	req *apiv1.ComputeHPImportanceRequest,
) (*apiv1.ComputeHPImportanceResponse, error) {
	experimentID := int(req.ExperimentId)
	if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, experimentID, false,
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	metricName := req.MetricName
	if metricName == "" {
		return nil, status.Error(codes.InvalidArgument, "must specify a metric name")
	}
	var metricType model.MetricType
	switch req.MetricType {
	case apiv1.MetricType_METRIC_TYPE_UNSPECIFIED:
		return nil, status.Error(codes.InvalidArgument, "must specify a metric type")
	case apiv1.MetricType_METRIC_TYPE_TRAINING:
		metricType = model.TrainingMetric
	case apiv1.MetricType_METRIC_TYPE_VALIDATION:
		metricType = model.ValidationMetric
	default:
		panic("Invalid metric type")
	}

	a.m.system.Ask(a.m.hpImportance, hpimportance.WorkRequest{
		ExperimentID: experimentID,
		MetricName:   metricName,
		MetricType:   metricType,
	})

	var resp apiv1.ComputeHPImportanceResponse
	return &resp, nil
}

// Translates MetricHPImportance to the protobuf form.
func protoMetricHPI(metricHpi model.MetricHPImportance,
) *apiv1.GetHPImportanceResponse_MetricHPImportance {
	return &apiv1.GetHPImportanceResponse_MetricHPImportance{
		Error:              metricHpi.Error,
		Pending:            metricHpi.Pending,
		InProgress:         metricHpi.InProgress,
		ExperimentProgress: metricHpi.ExperimentProgress,
		HpImportance:       metricHpi.HpImportance,
	}
}

func (a *apiServer) GetHPImportance(req *apiv1.GetHPImportanceRequest,
	resp apiv1.Determined_GetHPImportanceServer,
) error {
	experimentID := int(req.ExperimentId)
	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	var timeSinceLastAuth time.Time
	for {
		if time.Now().Sub(timeSinceLastAuth) >= recheckAuthPeriod {
			if _, _, err := a.getExperimentAndCheckCanDoActions(resp.Context(), experimentID, false,
				expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return err
			}
			timeSinceLastAuth = time.Now()
		}

		var response apiv1.GetHPImportanceResponse

		result, err := a.m.db.GetHPImportance(experimentID)
		if err != nil {
			return errors.Wrap(err, "error looking up hyperparameter importance")
		}
		response.TrainingMetrics = make(map[string]*apiv1.GetHPImportanceResponse_MetricHPImportance)
		response.ValidationMetrics = make(map[string]*apiv1.GetHPImportanceResponse_MetricHPImportance)
		for metric, metricHpi := range result.TrainingMetrics {
			response.TrainingMetrics[metric] = protoMetricHPI(metricHpi)
		}
		for metric, metricHpi := range result.ValidationMetrics {
			response.ValidationMetrics[metric] = protoMetricHPI(metricHpi)
		}

		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
		if err := resp.Send(&response); err != nil {
			return errors.Wrap(err, "error sending hyperparameter importance response")
		}

		allComplete := true
		if len(result.TrainingMetrics)+len(result.ValidationMetrics) == 0 {
			allComplete = false
		}
		for _, metricHpi := range result.TrainingMetrics {
			if metricHpi.Pending || metricHpi.InProgress {
				allComplete = false
			}
		}
		for _, metricHpi := range result.ValidationMetrics {
			if metricHpi.Pending || metricHpi.InProgress {
				allComplete = false
			}
		}
		if allComplete {
			state, _, err := a.m.db.GetExperimentStatus(experimentID)
			if err != nil {
				return errors.Wrap(err, "error looking up experiment state")
			}
			if model.TerminalStates[state] {
				return nil
			}
		}

		time.Sleep(period)
		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
	}
}

func (a *apiServer) GetBestSearcherValidationMetric(
	ctx context.Context, req *apiv1.GetBestSearcherValidationMetricRequest,
) (*apiv1.GetBestSearcherValidationMetricResponse, error) {
	if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId), false,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	metric, err := a.m.db.ExperimentBestSearcherValidation(int(req.ExperimentId))
	switch {
	case errors.Cause(err) == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "no validations for experiment")
	case err != nil:
		return nil, err
	}

	return &apiv1.GetBestSearcherValidationMetricResponse{
		Metric: metric,
	}, nil
}

func (a *apiServer) GetModelDef(
	ctx context.Context, req *apiv1.GetModelDefRequest,
) (*apiv1.GetModelDefResponse, error) {
	if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId), false,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	tgz, err := a.m.db.ExperimentModelDefinitionRaw(int(req.ExperimentId))
	if err != nil {
		return nil, errors.Wrapf(err,
			"error fetching model definition from database: %d", req.ExperimentId)
	}

	b64Tgz := base64.StdEncoding.EncodeToString(tgz)

	return &apiv1.GetModelDefResponse{B64Tgz: b64Tgz}, nil
}

func (a *apiServer) MoveExperiment(
	ctx context.Context, req *apiv1.MoveExperimentRequest,
) (*apiv1.MoveExperimentResponse, error) {
	// get experiment info
	exp, curUser, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId), false)
	if err != nil {
		return nil, err
	}
	if exp.Archived {
		return nil, errors.Errorf("experiment (%v) is archived and cannot be moved.", exp.ID)
	}

	// check that user can view destination project
	destProject, err := a.GetProjectByID(ctx, req.DestinationProjectId, curUser)
	if err != nil {
		return nil, err
	}
	if destProject.Archived {
		return nil, errors.Errorf("project (%v) is archived and cannot add new experiments.",
			req.DestinationProjectId)
	}

	// check that user can view source project
	srcProject, err := a.GetProjectByID(ctx, int32(exp.ProjectID), curUser)
	if err != nil {
		return nil, err
	}
	if srcProject.Archived {
		return nil, errors.Errorf("project (%v) is archived and cannot have experiments moved from it.",
			srcProject.Id)
	}

	if err = project.AuthZProvider.Get().CanMoveProjectExperiments(ctx, curUser, exp, srcProject,
		destProject); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	holder := &experimentv1.Experiment{}
	err = a.m.db.QueryProto("move_experiment", holder, req.ExperimentId,
		req.DestinationProjectId)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "experiment (%d) does not exist or not moveable by this user",
			req.ExperimentId)
	}

	return &apiv1.MoveExperimentResponse{},
		errors.Wrapf(err, "error moving experiment (%d)", req.ExperimentId)
}

func (a *apiServer) GetModelDefTree(
	ctx context.Context, req *apiv1.GetModelDefTreeRequest,
) (*apiv1.GetModelDefTreeResponse, error) {
	if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId), false,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	modelDefCache := GetModelDefCache()
	fileTree, err := modelDefCache.FileTreeNested(int(req.ExperimentId))
	if err != nil {
		return nil, err
	}
	return &apiv1.GetModelDefTreeResponse{Files: fileTree}, nil
}

func (a *apiServer) GetModelDefFile(
	ctx context.Context, req *apiv1.GetModelDefFileRequest,
) (*apiv1.GetModelDefFileResponse, error) {
	if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId), false,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	modelDefCache := GetModelDefCache()
	file, err := modelDefCache.FileContent(int(req.ExperimentId), req.Path)
	if err != nil {
		return nil, err
	}
	return &apiv1.GetModelDefFileResponse{File: file}, nil
}
