package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/hpimportance"
	"github.com/determined-ai/determined/master/internal/lttb"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

var experimentsAddr = actor.Addr("experiments")

func (a *apiServer) checkExperimentExists(id int) error {
	ok, err := a.m.db.CheckExperimentExists(id)
	switch {
	case err != nil:
		return status.Errorf(codes.Internal, "failed to check if experiment exists: %s", err)
	case !ok:
		return status.Errorf(codes.NotFound, "experiment %d not found", id)
	default:
		return nil
	}
}

func (a *apiServer) getExperiment(experimentID int) (*experimentv1.Experiment, error) {
	exp := &experimentv1.Experiment{}
	switch err := a.m.db.QueryProto("get_experiment", exp, experimentID); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "experiment not found: %d", experimentID)
	case err != nil:
		return nil, errors.Wrapf(err,
			"error fetching experiment from database: %d", experimentID)
	}
	return exp, nil
}

func (a *apiServer) GetExperiment(
	_ context.Context, req *apiv1.GetExperimentRequest,
) (*apiv1.GetExperimentResponse, error) {
	exp, err := a.getExperiment(int(req.ExperimentId))
	if err != nil {
		return nil, err
	}

	confBytes, err := a.m.db.ExperimentConfigRaw(int(req.ExperimentId))
	if err != nil {
		return nil, errors.Wrapf(err,
			"error fetching experiment config from database: %d", req.ExperimentId)
	}
	var conf map[string]interface{}
	err = json.Unmarshal(confBytes, &conf)
	if err != nil {
		return nil, errors.Wrapf(err,
			"error unmarshalling experiment config: %d", req.ExperimentId)
	}
	return &apiv1.GetExperimentResponse{Experiment: exp, Config: protoutils.ToStruct(conf)}, nil
}

func (a *apiServer) DeleteExperiment(
	_ context.Context, req *apiv1.DeleteExperimentRequest,
) (*apiv1.DeleteExperimentResponse, error) {
	// Avoid loading the experiment config for what may be a very old experiment.
	e, err := a.m.db.ExperimentWithoutConfigByID(int(req.ExperimentId))
	switch {
	case errors.Cause(err) == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "experiment not found")
	case err != nil:
		return nil, errors.Wrap(err, "failed to retrieve experiment")
	}

	switch exists, eErr := a.m.db.ExperimentHasCheckpointsInRegistry(int(req.ExperimentId)); {
	case eErr != nil:
		return nil, errors.New("failed to check model registry for references")
	case exists:
		return nil, status.Errorf(codes.InvalidArgument, "checkpoints are registered as model versions")
	}

	if !model.ExperimentTransitions[e.State][model.DeletingState] {
		return nil, fmt.Errorf("cannot delete experiment in %s state", e.State)
	}

	e.State = model.DeletingState
	if err := a.m.db.TrySaveExperimentState(e); err != nil {
		return nil, errors.Wrapf(err, "transitioning to %s", e.State)
	}
	go func() {
		if err := a.deleteExperiment(e); err != nil {
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

func (a *apiServer) deleteExperiment(exp *model.Experiment) error {
	conf, err := a.m.db.LegacyExperimentConfigByID(exp.ID)
	if err != nil {
		return fmt.Errorf("failed to read config for experiment: %w", err)
	}

	agentUserGroup, err := a.m.db.AgentUserGroup(*exp.OwnerID)
	switch {
	case err != nil:
		return errors.Errorf("cannot find user and group for experiment")
	case agentUserGroup == nil:
		agentUserGroup = &a.m.config.Security.DefaultTask
	}

	taskSpec := *a.m.taskSpec
	taskSpec.AgentUserGroup = agentUserGroup
	checkpoints, err := a.m.db.ExperimentCheckpointsToGCRaw(
		exp.ID,
		0,
		0,
		0,
		true,
	)
	if err != nil {
		return err
	}

	addr := actor.Addr(fmt.Sprintf("delete-checkpoint-gc-%s", uuid.New().String()))
	if gcErr := a.m.system.MustActorOf(addr, &checkpointGCTask{
		GCCkptSpec: tasks.GCCkptSpec{
			Base:               taskSpec,
			ExperimentID:       exp.ID,
			LegacyConfig:       conf,
			ToDelete:           checkpoints,
			DeleteTensorboards: true,
		},
		rm: a.m.rm,
		db: a.m.db,
	}).AwaitTermination(); gcErr != nil {
		return errors.Wrapf(gcErr, "failed to gc checkpoints for experiment")
	}

	trialIds, err := a.m.db.ExperimentTrialIDs(exp.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to gather trial IDs for experiment")
	}

	if err = a.m.trialLogBackend.DeleteTrialLogs(trialIds); err != nil {
		return errors.Wrapf(err, "failed to delete trial logs from backend")
	}

	if err = a.m.db.DeleteExperiment(exp.ID); err != nil {
		return errors.Wrapf(err, "deleting experiment from database")
	}
	return nil
}

func (a *apiServer) GetExperiments(
	_ context.Context, req *apiv1.GetExperimentsRequest) (*apiv1.GetExperimentsResponse, error) {
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
	orderColMap := map[apiv1.GetExperimentsRequest_SortBy]string{
		apiv1.GetExperimentsRequest_SORT_BY_UNSPECIFIED: "id",
		apiv1.GetExperimentsRequest_SORT_BY_ID:          "id",
		apiv1.GetExperimentsRequest_SORT_BY_DESCRIPTION: "description",
		apiv1.GetExperimentsRequest_SORT_BY_NAME:        "name",
		apiv1.GetExperimentsRequest_SORT_BY_START_TIME:  "start_time",
		apiv1.GetExperimentsRequest_SORT_BY_END_TIME:    "end_time",
		apiv1.GetExperimentsRequest_SORT_BY_STATE:       "state",
		apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS:  "num_trials",
		apiv1.GetExperimentsRequest_SORT_BY_PROGRESS:    "progress",
		apiv1.GetExperimentsRequest_SORT_BY_USER:        "username",
	}
	sortByMap := map[apiv1.OrderBy]string{
		apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
		apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
		apiv1.OrderBy_ORDER_BY_DESC:        "DESC",
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

	resp := &apiv1.GetExperimentsResponse{}
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
		req.Offset,
		req.Limit,
	)
}

func (a *apiServer) GetExperimentLabels(_ context.Context,
	req *apiv1.GetExperimentLabelsRequest) (*apiv1.GetExperimentLabelsResponse, error) {
	resp := &apiv1.GetExperimentLabelsResponse{}

	var err error
	labelUsage, err := a.m.db.ExperimentLabelUsage()
	if err != nil {
		return nil, err
	}

	// Convert the label usage map into a sorted list of labels
	// May add other sorting / pagination options later if needed
	labels := make([]string, len(labelUsage))
	i := 0
	for label := range labelUsage {
		labels[i] = label
		i++
	}
	sort.Slice(labels, func(i, j int) bool {
		return labelUsage[labels[i]] > labelUsage[labels[j]]
	})
	resp.Labels = labels

	return resp, nil
}

func (a *apiServer) GetExperimentValidationHistory(
	_ context.Context, req *apiv1.GetExperimentValidationHistoryRequest,
) (*apiv1.GetExperimentValidationHistoryResponse, error) {
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
	_ context.Context, req *apiv1.PreviewHPSearchRequest) (*apiv1.PreviewHPSearchResponse, error) {
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
	indexes := make(map[string]int)
	toProto := func(op searcher.ValidateAfter) ([]*experimentv1.RunnableOperation, error) {
		switch op.Length.Unit {
		case expconf.Records:
			return []*experimentv1.RunnableOperation{
				{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_VALIDATE,
					Length: &experimentv1.TrainingLength{
						Units:  experimentv1.TrainingLength_UNITS_RECORDS,
						Length: int32(op.Length.Units),
					},
				},
				{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_VALIDATE,
				},
			}, nil
		case expconf.Batches:
			return []*experimentv1.RunnableOperation{
				{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_TRAIN,
					Length: &experimentv1.TrainingLength{
						Units:  experimentv1.TrainingLength_UNITS_BATCHES,
						Length: int32(op.Length.Units),
					},
				},
				{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_VALIDATE,
				},
			}, nil
		case expconf.Epochs:
			return []*experimentv1.RunnableOperation{
				{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_TRAIN,
					Length: &experimentv1.TrainingLength{
						Units:  experimentv1.TrainingLength_UNITS_EPOCHS,
						Length: int32(op.Length.Units),
					},
				},
				{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_VALIDATE,
				},
			}, nil
		default:
			return []*experimentv1.RunnableOperation{},
				fmt.Errorf("unrecognized unit %s", op.Length.Unit)
		}
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
	if err = a.checkExperimentExists(int(req.Id)); err != nil {
		return nil, err
	}

	addr := experimentsAddr.Child(req.Id).String()
	switch err = a.actorRequest(addr, req, &resp); {
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
	if err = a.checkExperimentExists(int(req.Id)); err != nil {
		return nil, err
	}

	addr := experimentsAddr.Child(req.Id).String()
	switch err = a.actorRequest(addr, req, &resp); {
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
	if err = a.checkExperimentExists(int(req.Id)); err != nil {
		return nil, err
	}

	addr := experimentsAddr.Child(req.Id).String()
	err = a.actorRequest(addr, req, &resp)
	if status.Code(err) == codes.NotFound {
		return &apiv1.CancelExperimentResponse{}, nil
	}
	return resp, err
}

func (a *apiServer) KillExperiment(
	ctx context.Context, req *apiv1.KillExperimentRequest,
) (
	resp *apiv1.KillExperimentResponse, err error) {
	if err = a.checkExperimentExists(int(req.Id)); err != nil {
		return nil, err
	}

	addr := experimentsAddr.Child(req.Id).String()
	err = a.actorRequest(addr, req, &resp)
	if status.Code(err) == codes.NotFound {
		return &apiv1.KillExperimentResponse{}, nil
	}
	return resp, err
}

func (a *apiServer) ArchiveExperiment(
	ctx context.Context, req *apiv1.ArchiveExperimentRequest,
) (*apiv1.ArchiveExperimentResponse, error) {
	id := int(req.Id)

	dbExp, err := a.m.db.ExperimentWithoutConfigByID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "loading experiment %v", id)
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

	dbExp, err := a.m.db.ExperimentWithoutConfigByID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "loading experiment %v", id)
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

func (a *apiServer) PatchExperiment(
	ctx context.Context, req *apiv1.PatchExperimentRequest,
) (*apiv1.PatchExperimentResponse, error) {
	var exp experimentv1.Experiment
	switch err := a.m.db.QueryProto("get_experiment", &exp, req.Experiment.Id); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "experiment not found: %d", req.Experiment.Id)
	case err != nil:
		return nil, errors.Wrapf(err, "error fetching experiment from database: %d", req.Experiment.Id)
	}

	paths := req.UpdateMask.GetPaths()
	shouldUpdateNotes := false
	shouldUpdateConfig := false
	for _, path := range paths {
		switch {
		case path == "name":
			if len(strings.TrimSpace(req.Experiment.Name)) == 0 {
				return nil, status.Errorf(codes.InvalidArgument, "`name` is required.")
			}
			exp.Name = req.Experiment.Name
		case path == "notes":
			shouldUpdateNotes = true
			exp.Notes = req.Experiment.Notes
		case path == "labels":
			exp.Labels = req.Experiment.Labels
		case path == "description":
			exp.Description = req.Experiment.Description
		case !strings.HasPrefix(path, "update_mask"):
			return nil, status.Errorf(
				codes.InvalidArgument,
				"only 'name', 'notes', 'description', and 'labels' fields are mutable. cannot update %s", path)
		}
	}
	shouldUpdateConfig = (shouldUpdateNotes && len(paths) > 1) ||
		(!shouldUpdateNotes && len(paths) > 0)

	if shouldUpdateNotes {
		_, err := a.m.db.RawQuery(
			"patch_experiment_notes",
			req.Experiment.Id,
			req.Experiment.Notes,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update experiment")
		}
	}

	if shouldUpdateConfig {
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
		marshalledPatches, err := json.Marshal(patches)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal experiment patch")
		}
		_, err = a.m.db.RawQuery(
			"patch_experiment_config",
			req.Experiment.Id,
			marshalledPatches,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update experiment")
		}
	}

	return &apiv1.PatchExperimentResponse{Experiment: &exp}, nil
}

func (a *apiServer) GetExperimentCheckpoints(
	ctx context.Context, req *apiv1.GetExperimentCheckpointsRequest,
) (*apiv1.GetExperimentCheckpointsResponse, error) {
	ok, err := a.m.db.CheckExperimentExists(int(req.Id))
	switch {
	case err != nil:
		return nil, status.Errorf(codes.Internal, "failed to check if experiment exists: %s", err)
	case !ok:
		return nil, status.Errorf(codes.NotFound, "experiment %d not found", req.Id)
	}

	resp := &apiv1.GetExperimentCheckpointsResponse{}
	resp.Checkpoints = []*checkpointv1.Checkpoint{}
	switch err := a.m.db.QueryProto("get_checkpoints_for_experiment", &resp.Checkpoints, req.Id); {
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

		found = false
		for _, state := range req.ValidationStates {
			if state == v.ValidationState {
				found = true
				break
			}
		}

		if len(req.ValidationStates) != 0 && !found {
			return false
		}

		return true
	})

	a.sort(
		resp.Checkpoints, req.OrderBy, req.SortBy, apiv1.GetExperimentCheckpointsRequest_SORT_BY_TRIAL_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Checkpoints, req.Offset, req.Limit)
}

func (a *apiServer) CreateExperiment(
	ctx context.Context, req *apiv1.CreateExperimentRequest,
) (*apiv1.CreateExperimentResponse, error) {
	detParams := CreateExperimentParams{
		ConfigBytes:  req.Config,
		ModelDef:     filesToArchive(req.ModelDefinition),
		ValidateOnly: req.ValidateOnly,
	}
	if req.ParentId != 0 {
		parentID := int(req.ParentId)
		detParams.ParentID = &parentID
	}

	dbExp, validateOnly, taskSpec, err := a.m.parseCreateExperiment(&detParams)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid experiment: %s", err)
	}

	if validateOnly {
		return &apiv1.CreateExperimentResponse{}, nil
	}

	user, _, err := grpcutil.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	dbExp.OwnerID = &user.ID
	e, err := newExperiment(a.m, dbExp, taskSpec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create experiment: %s", err)
	}
	a.m.system.ActorOf(actor.Addr("experiments", e.ID), e)

	protoExp, err := a.getExperiment(e.ID)
	if err != nil {
		return nil, err
	}
	return &apiv1.CreateExperimentResponse{
		Experiment: protoExp, Config: protoutils.ToStruct(e.Config),
	}, nil
}

var defaultMetricsStreamPeriod = 30 * time.Second

func (a *apiServer) MetricNames(req *apiv1.MetricNamesRequest,
	resp apiv1.Determined_MetricNamesServer) error {
	experimentID := int(req.ExperimentId)
	if err := a.checkExperimentExists(experimentID); err != nil {
		return err
	}
	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	config, err := a.m.db.ExperimentConfig(experimentID)
	if err != nil {
		return errors.Wrapf(err,
			"error fetching experiment config from database: %d", experimentID)
	}
	searcherMetric := config.Searcher().Metric()

	seenTrain := make(map[string]bool)
	seenValid := make(map[string]bool)
	var tStartTime time.Time
	var vStartTime time.Time
	for {
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

func (a *apiServer) MetricBatches(req *apiv1.MetricBatchesRequest,
	resp apiv1.Determined_MetricBatchesServer) error {
	experimentID := int(req.ExperimentId)
	if err := a.checkExperimentExists(experimentID); err != nil {
		return err
	}
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

	seenBatches := make(map[int32]bool)
	var startTime time.Time
	for {
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
	resp apiv1.Determined_TrialsSnapshotServer) error {
	experimentID := int(req.ExperimentId)
	if err := a.checkExperimentExists(experimentID); err != nil {
		return err
	}
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

	var startTime time.Time
	for {
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
	trials []int32, err error) {
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
	case expconf.AsyncHalvingConfig:
		ranking = ByTrainingLength
	case expconf.AdaptiveASHAConfig:
		ranking = ByTrainingLength
	case expconf.SingleConfig:
		return nil, errors.New("single-trial experiments are not supported for trial sampling")
	case expconf.PBTConfig:
		return nil, errors.New("population-based training not supported for trial sampling")
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
	trialCursors map[int32]time.Time) (*apiv1.TrialsSampleResponse_Trial, error) {
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
		metricSeries = lttb.Downsample(metricSeries, maxDatapoints)
	}

	for _, in := range metricSeries {
		out := apiv1.TrialsSampleResponse_DataPoint{
			Batches: int32(in.X),
			Value:   in.Y,
		}
		trial.Data = append(trial.Data, &out)
	}
	return &trial, nil
}

func (a *apiServer) TrialsSample(req *apiv1.TrialsSampleRequest,
	resp apiv1.Determined_TrialsSampleServer) error {
	experimentID := int(req.ExperimentId)
	if err := a.checkExperimentExists(experimentID); err != nil {
		return err
	}
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

	config, err := a.m.db.ExperimentConfig(experimentID)
	if err != nil {
		return errors.Wrapf(err, "error fetching experiment config from database")
	}
	searcherConfig := config.Searcher()

	trialCursors := make(map[int32]time.Time)
	currentTrials := make(map[int32]bool)
	for {
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

func (a *apiServer) ComputeHPImportance(ctx context.Context,
	req *apiv1.ComputeHPImportanceRequest) (*apiv1.ComputeHPImportanceResponse, error) {
	experimentID := int(req.ExperimentId)
	if err := a.checkExperimentExists(experimentID); err != nil {
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
	resp apiv1.Determined_GetHPImportanceServer) error {
	experimentID := int(req.ExperimentId)
	if err := a.checkExperimentExists(experimentID); err != nil {
		return err
	}

	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	for {
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
	_ context.Context, req *apiv1.GetBestSearcherValidationMetricRequest,
) (*apiv1.GetBestSearcherValidationMetricResponse, error) {
	if err := a.checkExperimentExists(int(req.ExperimentId)); err != nil {
		return nil, err
	}

	metric, err := a.m.db.ExperimentBestSearcherValidation(int(req.ExperimentId))
	if err != nil {
		return nil, err
	}

	return &apiv1.GetBestSearcherValidationMetricResponse{
		Metric: metric,
	}, nil
}
