package internal

import (
	"context"
	"encoding/json"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
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

func (a *apiServer) ActivateExperiment(
	ctx context.Context, req *apiv1.ActivateExperimentRequest) (
	resp *apiv1.ActivateExperimentResponse, err error) {

	var rawExp []byte
	var exp model.Experiment
	if rawExp, err = a.m.db.ExperimentRaw(int(req.Id)); err != nil {
		return nil, status.Errorf(codes.NotFound, "%s; experiment %d not found.", err.Error(), req.Id)
	}
	if err = json.Unmarshal(rawExp, &exp); err != nil {
		return nil, status.Error(codes.Internal, "failed to unmarshal experiment")
	}
	if model.TerminalStates[exp.State] {
		return nil, status.Errorf(codes.InvalidArgument, "experiment in incompatible state: %s",
			exp.State)
	}

	addr := actor.Addr("experiments", req.Id).String()
	err = a.actorRequest(addr, req, &resp)
	return resp, err
}

func (a *apiServer) PauseExperiment(
	ctx context.Context, req *apiv1.PauseExperimentRequest) (
	resp *apiv1.PauseExperimentResponse, err error) {

	var rawExp []byte
	var exp model.Experiment
	if rawExp, err = a.m.db.ExperimentRaw(int(req.Id)); err != nil {
		return nil, status.Errorf(codes.NotFound, "%s; experiment %d not found.", err.Error(), req.Id)
	}
	if err = json.Unmarshal(rawExp, &exp); err != nil {
		return nil, status.Error(codes.Internal, "failed to unmarshal experiment")
	}
	if model.TerminalStates[exp.State] {
		return nil, status.Errorf(codes.InvalidArgument, "experiment in incompatible state: %s",
			exp.State)
	}

	addr := actor.Addr("experiments", req.Id).String()
	err = a.actorRequest(addr, req, &resp)
	return resp, err
}

func (a *apiServer) CancelExperiment(
	ctx context.Context, req *apiv1.CancelExperimentRequest) (
	resp *apiv1.CancelExperimentResponse, err error) {

	var rawExp []byte
	var exp model.Experiment
	if rawExp, err = a.m.db.ExperimentRaw(int(req.Id)); err != nil {
		return nil, status.Errorf(codes.NotFound, "%s; experiment %d not found.", err.Error(), req.Id)
	}
	if err = json.Unmarshal(rawExp, &exp); err != nil {
		return nil, status.Error(codes.Internal, "failed to unmarshal experiment")
	}
	if model.TerminalStates[exp.State] {
		return &apiv1.CancelExperimentResponse{}, nil
	}

	addr := actor.Addr("experiments", req.Id).String()
	err = a.actorRequest(addr, req, &resp)
	return resp, err
}

func (a *apiServer) KillExperiment(
	ctx context.Context, req *apiv1.KillExperimentRequest) (
	resp *apiv1.KillExperimentResponse, err error) {

	var rawExp []byte
	var exp model.Experiment
	if rawExp, err = a.m.db.ExperimentRaw(int(req.Id)); err != nil {
		return nil, status.Errorf(codes.NotFound, "%s; experiment %d not found.", err.Error(), req.Id)
	}
	if err = json.Unmarshal(rawExp, &exp); err != nil {
		return nil, status.Error(codes.Internal, "failed to unmarshal experiment")
	}
	if model.TerminalStates[exp.State] {
		return &apiv1.KillExperimentResponse{}, nil
	}

	addr := actor.Addr("experiments", req.Id).String()
	err = a.actorRequest(addr, req, &resp)
	return resp, err
}

func (a *apiServer) setArchiveStatus(reqID int32, doArchive bool) (
	*experimentv1.Experiment, error) {
	exp := experimentv1.Experiment{}
	err := a.m.db.QueryProto("set_experiment_archive", &exp, reqID, doArchive)
	return &exp, err
}

func (a *apiServer) ArchiveExperiment(
	ctx context.Context, req *apiv1.ArchiveExperimentRequest) (
	*apiv1.ArchiveExperimentResponse, error) {
	exp, err := a.setArchiveStatus(req.Id, true)
	return &apiv1.ArchiveExperimentResponse{Experiment: exp}, err
}

func (a *apiServer) UnarchiveExperiment(
	ctx context.Context, req *apiv1.UnarchiveExperimentRequest) (
	*apiv1.UnarchiveExperimentResponse, error) {
	exp, err := a.setArchiveStatus(req.Id, false)
	return &apiv1.UnarchiveExperimentResponse{Experiment: exp}, err
}
