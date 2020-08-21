package internal

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/stepv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

const (
	batchSize     = 1000
	batchWaitTime = 100 * time.Millisecond
)

func trialStatus(d *db.PgDB, trialID int32) (model.State, int, error) {
	trialStatus := struct {
		State   model.State
		NumLogs int
	}{}
	err := d.Query("trial_status", &trialStatus, trialID)
	if err == db.ErrNotFound {
		err = status.Error(codes.NotFound, "trial not found")
	}
	return trialStatus.State, trialStatus.NumLogs, err
}

func (a *apiServer) TrialLogs(
	req *apiv1.TrialLogsRequest, resp apiv1.Determined_TrialLogsServer) error {
	if err := grpc.ValidateRequest(
		grpc.ValidateLimit(req.Limit),
	); err != nil {
		return err
	}
	_, total, err := trialStatus(a.m.db, req.TrialId)
	if err != nil {
		return err
	}

	offset := effectiveOffset(int(req.Offset), total)

	if limit := int32(total - offset); !req.Follow && (limit < req.Limit || req.Limit == 0) {
		req.Limit = limit
	}
	count := 0
	for {
		queryLimit := int(req.Limit) - count
		if req.Limit == 0 || queryLimit > batchSize {
			queryLimit = batchSize
		}
		if queryLimit <= 0 {
			return nil
		}
		var logs []*apiv1.TrialLogsResponse
		if err := a.m.db.QueryProto("stream_logs", &logs, req.TrialId, offset, queryLimit); err != nil {
			return err
		}
		for i, log := range logs {
			log.Id = int32(offset + i)
			if err := resp.Send(log); err != nil {
				return err
			}
		}
		newRecords := len(logs)
		count += newRecords
		offset += newRecords
		if newRecords < queryLimit {
			state, _, err := trialStatus(a.m.db, req.TrialId)
			if err != nil || model.TerminalStates[state] {
				return err
			}
		}
		time.Sleep(batchWaitTime)
	}
}

func (a *apiServer) GetTrialCheckpoints(
	_ context.Context, req *apiv1.GetTrialCheckpointsRequest,
) (*apiv1.GetTrialCheckpointsResponse, error) {
	_, _, err := trialStatus(a.m.db, req.Id)
	if err != nil {
		return nil, err
	}

	resp := &apiv1.GetTrialCheckpointsResponse{}
	resp.Checkpoints = []*checkpointv1.Checkpoint{}

	switch err := a.m.db.QueryProto("get_checkpoints_for_trial", &resp.Checkpoints, req.Id); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "no checkpoints found for trial %d", req.Id)
	case err != nil:
		return nil,
			errors.Wrapf(err, "error fetching checkpoints for trial %d from database", req.Id)
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
		resp.Checkpoints, req.OrderBy, req.SortBy, apiv1.GetTrialCheckpointsRequest_SORT_BY_BATCH_NUMBER)

	return resp, a.paginate(&resp.Pagination, &resp.Checkpoints, req.Offset, req.Limit)
}

func (a *apiServer) KillTrial(
	ctx context.Context, req *apiv1.KillTrialRequest,
) (*apiv1.KillTrialResponse, error) {
	ok, err := a.m.db.CheckTrialExists(int(req.Id))
	switch {
	case err != nil:
		return nil, status.Errorf(codes.Internal, "failed to check if trial exists: %s", err)
	case !ok:
		return nil, status.Errorf(codes.NotFound, "trial %d not found", req.Id)
	}

	resp := apiv1.KillTrialResponse{}
	addr := actor.Addr("trials", req.Id).String()
	err = a.actorRequest(addr, req, &resp)
	if status.Code(err) == codes.NotFound {
		return &apiv1.KillTrialResponse{}, nil
	}
	return &resp, err
}

// func trialToProcessedLength() {

// 	// addr := actor.Addr("trials", req.Id).String()
// 	// switch err = a.actorRequest(addr, req, &resp); {
// }

func (a *apiServer) GetExperimentTrials(
	_ context.Context, req *apiv1.GetExperimentTrialsRequest) (*apiv1.GetExperimentTrialsResponse, error) {
	resp := &apiv1.GetExperimentTrialsResponse{}

	switch err := a.m.db.QueryProto("get_trials_for_experiment", &resp.Trials, req.ExperimentId); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "experiment %d not found:", req.ExperimentId)
	case err != nil:
		return nil, err
	}
	a.filter(&resp.Trials, func(i int) bool {
		v := resp.Trials[i]
		eliminate := false

		if len(req.States) != 0 && !eliminate {
			eliminate = true
			for _, state := range req.States {
				if state == v.State {
					eliminate = false
					break
				}
			}
		}

		return !eliminate
	})

	a.sort(resp.Trials, req.OrderBy, req.SortBy, apiv1.GetExperimentTrialsRequest_SORT_BY_ID)
	if err := a.paginate(&resp.Pagination, &resp.Trials, req.Offset, req.Limit); err != nil {
		return nil, err
	}

	// expConfig, err := a.m.db.ExperimentConfig(int(req.ExperimentId))
	// if err != nil {
	// 	return nil, err
	// }

	// type valCheckpoint struct {
	// 	BestValidation   float64 `json:"best_validation"`
	// 	LatestValidation float64 `json:"latest_validation"`
	// 	BestCheckpoint   string  `json:"best_checkpoint"`
	// }

	// for _, trial := range resp.Trials {
	// 	// TODO consider missing (eg terminal) trials
	// 	trialAddr := actor.Addr("trials", trial.Id)
	// 	var tbp int
	// 	// OPT do in parallel?

	// 	resp := a.m.system.AskAt(trialAddr, trialProgress{})
	// 	switch {
	// 	case resp.Empty():
	// 		// status.Errorf(codes.NotFound, "/api/v1%s not found", addr)
	// 		fmt.Println("no active actor")
	// 		// FIXME how do I calculate this
	// 	case resp.Error() != nil:
	// 		return nil, resp.Error()
	// 	default:
	// 		reflect.ValueOf(tbp).Elem().Set(reflect.ValueOf(resp.Get()))
	// 		trial.ProcessedLength = &trialv1.Length{
	// 			Value: int32(tbp),
	// 			Unit:  trialv1.LengthUnit_LENGTH_UNIT_BATCHES,
	// 		}

	// 	}

	// var stats valCheckpoint
	// res, err := a.m.db.RawQuery("get_trial_stats", req.ExperimentId, trial.Id)
	// if err != nil {
	// 	return nil, err
	// }
	// fmt.Println(res)
	// if err = json.Unmarshal(res, &stats); err != nil {
	// 	return nil, err
	// }

	// // TODO populate the whole checkpoint
	// trial.BestCheckpoint = &checkpointv1.Checkpoint{Uuid: stats.BestCheckpoint}
	// trial.BestValidation = stats.BestValidation
	// trial.LatestValidation = stats.LatestValidation
	// }

	return resp, nil
}

// this could basically be a special case of getTrials assuming we support include_steps query param
func (a *apiServer) GetTrial(_ context.Context, req *apiv1.GetTrialRequest) (
	*apiv1.GetTrialResponse, error,
) {
	var protoTrial trialv1.Trial
	var response apiv1.GetTrialResponse
	if err := a.m.db.QueryProto("get_prototrial", &protoTrial, req.Id); err != nil {
		return nil, err
	}
	response.Trial = &protoTrial
	if req.IncludeSteps {
		var protoSteps []*stepv1.Step
		// OPT merge with first query for better performance.
		if err := a.m.db.QueryProto("get_trial_protosteps", &protoSteps); err != nil {
			return nil, err
		}
		response.Steps = protoSteps
	}
	return &response, nil
}
