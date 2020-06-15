package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

type dbExperiment struct {
	ID          int32
	Description string
	Labels      labels
	StartTime   time.Time
	EndTime     *time.Time
	State       model.State
	NumTrials   int
	Archived    bool
	Progress    float64
	Username    string
}

func toProtoExperiment(exp dbExperiment) *experimentv1.Experiment {
	return &experimentv1.Experiment{
		Id:          exp.ID,
		Description: exp.Description,
		Labels:      exp.Labels,
		StartTime: &timestamppb.Timestamp{
			Seconds: exp.StartTime.Unix(),
			Nanos:   int32(exp.StartTime.Nanosecond()),
		},
		EndTime: &timestamppb.Timestamp{
			Seconds: exp.StartTime.Unix(),
			Nanos:   int32(exp.StartTime.Nanosecond()),
		},
		State:     toProtoState(exp.State),
		NumTrials: int32(exp.NumTrials),
		Archived:  exp.Archived,
		Progress:  exp.Progress,
		Username:  exp.Username,
	}
}

func toProtoState(state model.State) experimentv1.State {
	switch state {
	case model.ActiveState:
		return experimentv1.State_STATE_ACTIVE
	case model.CanceledState:
		return experimentv1.State_STATE_CANCELED
	case model.CompletedState:
		return experimentv1.State_STATE_COMPLETED
	case model.DeletedState:
		return experimentv1.State_STATE_DELETED
	case model.ErrorState:
		return experimentv1.State_STATE_ERROR
	case model.PausedState:
		return experimentv1.State_STATE_PAUSED
	case model.StoppingCanceledState:
		return experimentv1.State_STATE_STOPPING_CANCELED
	case model.StoppingCompletedState:
		return experimentv1.State_STATE_STOPPING_COMPLETED
	case model.StoppingErrorState:
		return experimentv1.State_STATE_STOPPING_ERROR
	default:
		return experimentv1.State_STATE_UNSPECIFIED
	}
}

type labels []string

func (l *labels) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	data, ok := src.([]byte)
	if !ok {
		return errors.Errorf("unable to convert to []byte: %v", src)
	}
	var labels []string
	if err := json.Unmarshal(data, &labels); err != nil {
		return err
	}
	*l = labels
	return nil
}

func (a *apiServer) GetExperiments(
	_ context.Context, req *apiv1.GetExperimentsRequest) (*apiv1.GetExperimentsResponse, error) {
	resp := &apiv1.GetExperimentsResponse{}
	var values []dbExperiment
	err := a.m.db.Query("get_experiments", &values)
	if err != nil {
		return nil, err
	}
	for _, exp := range values {
		exp := toProtoExperiment(exp)
		if filterExperiments(req, exp) {
			resp.Experiments = append(resp.Experiments, exp)
		}
	}
	resp.Pagination, err = a.pagination(resp.Experiments, req.Offset, req.Limit)
	if err != nil {
		return nil, err
	}
	sort.Slice(resp.Experiments, func(i, j int) bool {
		a1, a2 := resp.Experiments[i], resp.Experiments[j]
		if req.OrderBy == apiv1.OrderBy_ORDER_BY_DESC {
			a1, a2 = a2, a1
		}
		switch req.SortBy {
		case apiv1.GetExperimentsRequest_SORT_BY_ID, apiv1.GetExperimentsRequest_SORT_BY_UNSPECIFIED:
			return a1.Id < a2.Id
		case apiv1.GetExperimentsRequest_SORT_BY_DESCRIPTION:
			return a1.Description < a2.Description
		case apiv1.GetExperimentsRequest_SORT_BY_START_TIME:
			return a1.StartTime.Seconds < a2.StartTime.Seconds
		case apiv1.GetExperimentsRequest_SORT_BY_END_TIME:
			switch {
			case a1.EndTime == nil && a2.EndTime == nil:
				return a1.Id < a2.Id
			case a1.EndTime == nil:
				return false
			case a2.EndTime == nil:
				return true
			default:
				return a1.EndTime.Seconds < a2.EndTime.Seconds
			}
		case apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS:
			return a1.NumTrials < a2.NumTrials
		case apiv1.GetExperimentsRequest_SORT_BY_STATE:
			return a1.State < a2.State
		case apiv1.GetExperimentsRequest_SORT_BY_PROGRESS:
			return a1.Progress < a2.Progress
		case apiv1.GetExperimentsRequest_SORT_BY_USER:
			return a1.Username < a2.Username
		default:
			panic(fmt.Sprintf("unknown sort type specified: %s", req.SortBy))
		}
	})
	resp.Experiments = resp.Experiments[resp.Pagination.StartIndex:resp.Pagination.EndIndex]
	return resp, nil
}

func filterExperiments(req *apiv1.GetExperimentsRequest, exp *experimentv1.Experiment) bool {
	if req.Archived != nil && req.Archived.Value != exp.Archived {
		return false
	}
	if len(req.States) > 0 {
		found := false
		for _, state := range req.States {
			if exp.State == state {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	if len(req.Users) > 0 {
		found := false
		for _, user := range req.Users {
			if exp.Username == user {
				found = true
			}
		}
		if !found {
			return false
		}
		return false
	}
	return strings.Contains(strings.ToLower(exp.Description), strings.ToLower(req.Description))
}
