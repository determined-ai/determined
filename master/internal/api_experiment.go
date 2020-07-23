package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

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
	toProto := func(op searcher.Runnable) experimentv1.RunnableOperation {
		switch op := op.(type) {
		case searcher.Train:
			switch op.Length.Unit {
			case model.Records:
				return experimentv1.RunnableOperation{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_TRAIN,
					Length: &experimentv1.TrainingLength{
						Unit:  experimentv1.Unit_UNIT_RECORDS,
						Units: int32(op.Length.Units),
					},
				}
			case model.Batches:
				return experimentv1.RunnableOperation{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_TRAIN,
					Length: &experimentv1.TrainingLength{
						Unit:  experimentv1.Unit_UNIT_BATCHES,
						Units: int32(op.Length.Units),
					},
				}
			case model.Epochs:
				return experimentv1.RunnableOperation{
					Type: experimentv1.RunnableType_RUNNABLE_TYPE_TRAIN,
					Length: &experimentv1.TrainingLength{
						Unit:  experimentv1.Unit_UNIT_EPOCHS,
						Units: int32(op.Length.Units),
					},
				}
			default:
				return experimentv1.RunnableOperation{
					Type:   experimentv1.RunnableType_RUNNABLE_TYPE_TRAIN,
					Length: &experimentv1.TrainingLength{Unit: experimentv1.Unit_UNIT_UNSPECIFIED},
				}
			}
		case searcher.Validate:
			return experimentv1.RunnableOperation{
				Type: experimentv1.RunnableType_RUNNABLE_TYPE_VALIDATE,
			}
		case searcher.Checkpoint:
			return experimentv1.RunnableOperation{
				Type: experimentv1.RunnableType_RUNNABLE_TYPE_CHECKPOINT,
			}
		default:
			return experimentv1.RunnableOperation{
				Type: experimentv1.RunnableType_RUNNABLE_TYPE_UNSPECIFIED,
			}
		}
	}
	for _, result := range sim.Results {
		var operations []*experimentv1.RunnableOperation
		for _, msg := range result {
			op := toProto(msg)
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
