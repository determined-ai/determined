package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

func listToMapSet(list *[]string) *map[string]bool {
	mapSet := make(map[string]bool)
	for _, key := range *list {
		mapSet[key] = true
	}
	return &mapSet
}

func mapSetToList(mapSet *map[string]bool) *[]string {
	var list []string
	for key, value := range *mapSet {
		if value {
			list = append(list, key)
		}
	}
	return &list
}

// CHECK don't we do this anywhere else?
func modelExpToProtoV1Experiment(exp *model.Experiment) *experimentv1.Experiment {
	labels := map[string]bool(exp.Config.Labels) // CHECK there is gotta be a better way
	return &experimentv1.Experiment{
		Description: exp.Config.Description,
		Id:          int32(exp.ID),
		Labels:      *mapSetToList(&labels),
	}
	// TODO finish the translation
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

func (a *apiServer) GetExperiment(
	_ context.Context, req *apiv1.GetExperimentRequest,
) (*apiv1.GetExperimentResponse, error) {
	exp := &experimentv1.Experiment{}
	switch err := a.m.db.QueryProto("get_experiment", exp, req.ExperimentId); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "experiment not found: %d", req.ExperimentId)
	case err != nil:
		return nil, errors.Wrapf(err,
			"error fetching experiment from database: %d", req.ExperimentId)
	}

	conf, err := a.m.db.ExperimentConfig(int(req.ExperimentId))
	if err != nil {
		return nil, errors.Wrapf(err,
			"error fetching experiment config from database: %d", req.ExperimentId)
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
		return strings.Contains(strings.ToLower(v.Description), strings.ToLower(req.Description))
	})
	a.sort(resp.Experiments, req.OrderBy, req.SortBy, apiv1.GetExperimentsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Experiments, req.Offset, req.Limit)
}

func (a *apiServer) PreviewHPSearch(
	_ context.Context, req *apiv1.PreviewHPSearchRequest) (*apiv1.PreviewHPSearchResponse, error) {
	bytes, err := protojson.Marshal(req.Config)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error parsing experiment config: %s", err)
	}
	config := model.DefaultExperimentConfig()
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

func (a *apiServer) SetExperimentDescription(
	ctx context.Context, req *apiv1.SetExperimentDescriptionRequest,
) (*apiv1.SetExperimentDescriptionResponse, error) {
	id := int(req.Id)

	dbExp, err := a.m.db.ExperimentByID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "loading experiment %v", id)
	}

	dbExp.Config.Description = req.Description
	err = a.m.db.SaveExperimentConfig(dbExp)
	switch err {
	case nil:
		return &apiv1.SetExperimentDescriptionResponse{}, nil
	default:
		return nil, errors.Wrapf(err, "failed to save experiment %d config",
			req.Id)
	}
}

func (a *apiServer) SetExperimentLabels(
	ctx context.Context, req *apiv1.SetExperimentLabelsRequest,
) (*apiv1.SetExperimentLabelsResponse, error) {
	id := int(req.Id)

	dbExp, err := a.m.db.ExperimentByID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "loading experiment %v", id)
	}

	labelsMap := make(map[string]bool)
	for _, label := range req.Labels {
		labelsMap[label] = true
	}
	dbExp.Config.Labels = labelsMap
	err = a.m.db.SaveExperimentConfig(dbExp)
	switch err {
	case nil:
		return &apiv1.SetExperimentLabelsResponse{}, nil
	default:
		return nil, errors.Wrapf(err, "failed to save experiment %d config",
			req.Id)
	}
}

// rename
func (a *apiServer) UpdateExperiment(
	ctx context.Context, req *apiv1.UpdateExperimentRequest,
) (*experimentv1.Experiment, error) {

	fmt.Printf("description: %s\n", req.Experiment.Description)

	// DISCUSS or we could start with a experimentv1 query.
	dbExp, err := a.m.db.ExperimentByID(int(req.Experiment.Id))
	if err != nil {
		return nil, errors.Wrapf(err, "loading experiment %v", req.Experiment.Id)
	}

	paths := req.UpdateMask.GetPaths()
	fmt.Printf("paths: %v \n", paths)
	for _, path := range paths {
		switch {
		case path == "description":
			dbExp.Config.Description = req.Experiment.Description
		case path == "labels":
			dbExp.Config.Labels = *listToMapSet(&req.Experiment.Labels)
		case !strings.HasPrefix(path, "update_mask"):
			return nil, status.Errorf(
				codes.InvalidArgument,
				"only description and labels fields are mutable. cannot update %s", path)
		}
	}

	err = a.m.db.SaveExperimentConfig(dbExp)
	switch err {
	case nil:
		return modelExpToProtoV1Experiment(dbExp), nil
	default:
		return nil, errors.Wrapf(err, "failed to save experiment %d config",
			req.Experiment.Id)
	}
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
