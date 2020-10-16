package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

func isInList(srcList []string, item string) bool {
	item = strings.ToLower(item)
	for _, src := range srcList {
		if strings.Contains(strings.ToLower(src), item) {
			return true
		}
	}
	return false
}

// matchesList checks whether srcList contains all strings provided in matchList.
func matchesList(srcList []string, matchList []string) bool {
	for _, match := range matchList {
		if !isInList(srcList, match) {
			return false
		}
	}

	return true
}

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

func (a *apiServer) GetExperiments(
	_ context.Context, req *apiv1.GetExperimentsRequest) (*apiv1.GetExperimentsResponse, error) {
	resp := &apiv1.GetExperimentsResponse{}
	if err := a.m.db.QueryProto("get_experiments", &resp.Experiments); err != nil {
		return nil, err
	}
	a.filter(&resp.Experiments, func(i int) bool {
		v := resp.Experiments[i]
		if req.Archived != nil && req.Archived.Value != v.Archived {
			return false
		}
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
		for _, user := range req.Users {
			if user == v.Username {
				found = true
				break
			}
		}
		if len(req.Users) != 0 && !found {
			return false
		}

		if !matchesList(v.Labels, req.Labels) {
			return false
		}

		return strings.Contains(strings.ToLower(v.Description), strings.ToLower(req.Description))
	})
	a.sort(resp.Experiments, req.OrderBy, req.SortBy, apiv1.GetExperimentsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Experiments, req.Offset, req.Limit)
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
	config := model.DefaultExperimentConfig(&a.m.config.TaskContainerDefaults)
	if err = json.Unmarshal(bytes, &config); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error parsing experiment config: %s", err)
	}
	if err = check.Validate(config.Searcher); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid experiment config: %s", err)
	}

	sm := searcher.NewSearchMethod(config.Searcher)
	s := searcher.NewSearcher(req.Seed, sm, config.Hyperparameters)
	sim, err := searcher.Simulate(s, nil, searcher.RandomValidation, true, config.Searcher.Metric)
	if err != nil {
		return nil, err
	}
	protoSim := &experimentv1.ExperimentSimulation{Seed: req.Seed}
	indexes := make(map[string]int)
	toProto := func(op searcher.Runnable) (experimentv1.RunnableOperation, error) {
		switch op := op.(type) {
		case searcher.Train:
			switch op.Length.Unit {
			case model.Records:
				return experimentv1.RunnableOperation{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_TRAIN,
					Length: &experimentv1.TrainingUnits{
						Unit:  experimentv1.Unit_UNIT_RECORDS,
						Count: int32(op.Length.Units),
					},
				}, nil
			case model.Batches:
				return experimentv1.RunnableOperation{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_TRAIN,
					Length: &experimentv1.TrainingUnits{
						Unit:  experimentv1.Unit_UNIT_BATCHES,
						Count: int32(op.Length.Units),
					},
				}, nil
			case model.Epochs:
				return experimentv1.RunnableOperation{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_TRAIN,
					Length: &experimentv1.TrainingUnits{
						Unit:  experimentv1.Unit_UNIT_EPOCHS,
						Count: int32(op.Length.Units),
					},
				}, nil
			default:
				return experimentv1.RunnableOperation{},
					fmt.Errorf("unrecognized unit %s", op.Length.Unit)
			}
		case searcher.Validate:
			return experimentv1.RunnableOperation{
				Type: experimentv1.RunnableType_RUNNABLE_TYPE_VALIDATE,
			}, nil
		case searcher.Checkpoint:
			return experimentv1.RunnableOperation{
				Type: experimentv1.RunnableType_RUNNABLE_TYPE_CHECKPOINT,
			}, nil
		default:
			return experimentv1.RunnableOperation{},
				fmt.Errorf("unrecognized searcher.Runnable %s", op)
		}
	}
	for _, result := range sim.Results {
		var operations []*experimentv1.RunnableOperation
		for _, msg := range result {
			op, err := toProto(msg)
			if err != nil {
				return nil, errors.Wrapf(err, "error converting msg in simultion result %s", msg)
			}
			operations = append(operations, &op)
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

	addr := actor.Addr("experiments", req.Id).String()
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

	addr := actor.Addr("experiments", req.Id).String()
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

	addr := actor.Addr("experiments", req.Id).String()
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

	addr := actor.Addr("experiments", req.Id).String()
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
	for _, path := range paths {
		switch {
		case path == "description":
			exp.Description = req.Experiment.Description
		case path == "labels":
			exp.Labels = req.Experiment.Labels
		case !strings.HasPrefix(path, "update_mask"):
			return nil, status.Errorf(
				codes.InvalidArgument,
				"only description and labels fields are mutable. cannot update %s", path)
		}
	}

	type experimentPatch struct {
		Labels      []string `json:"labels"`
		Description string   `json:"description"`
	}
	patches := experimentPatch{Description: exp.Description, Labels: exp.Labels}
	marshalledPatches, err := json.Marshal(patches)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal experiment patches")
	}

	if _, err := a.m.db.RawQuery(
		"patch_experiment",
		req.Experiment.Id,
		marshalledPatches,
	); err != nil {
		return nil, errors.Wrapf(err, "error updating experiment in database: %d", req.Experiment.Id)
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

// TODO rename me.
func reqToP(
	req *apiv1.PostExperimentRequest,
) (*PostExperimentParams, error) {

	params := PostExperimentParams{
		ConfigBytes:  protojson.Format(req.Config),
		ModelDef:     filesToArchive(req.Context),
		ValidateOnly: req.ValidateOnly,
	}
	if req.ParentId != 0 {
		parentID := int(req.ParentId)
		params.ParentID = &parentID
	}
	return &params, nil
}

func (a *apiServer) PostExperiment(
	ctx context.Context, req *apiv1.PostExperimentRequest,
) (*apiv1.PostExperimentResponse, error) {
	detParams, err := reqToP(req)
	if err != nil {
		return nil, err
	}
	dbExp, validateOnly, err := a.m.postParseExperiment(detParams)

	if err != nil {
		return nil, echo.NewHTTPError(
			http.StatusBadRequest,
			errors.Wrap(err, "invalid experiment"))
	}

	if validateOnly {
		// FIXME. in the old api do we return an error here?
		return nil, status.Errorf(codes.OK, "it's fine")
	}

	user, _, err := grpc.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	dbExp.OwnerID = &user.ID
	e, err := newExperiment(a.m, dbExp)
	if err != nil {
		return nil, errors.Wrap(err, "starting experiment")
	}
	a.m.system.ActorOf(actor.Addr("experiments", e.ID), e)

	// DISCUSS if we could convert model.Experiment to proto experiment in memory..
	protoExp, err := a.getExperiment(dbExp.ID)
	if err != nil {
		return nil, err
	}
	return &apiv1.PostExperimentResponse{Experiment: protoExp}, nil
}
