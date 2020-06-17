package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

func (a *apiServer) GetExperiments(
	_ context.Context, req *apiv1.GetExperimentsRequest) (*apiv1.GetExperimentsResponse, error) {
	resp := &apiv1.GetExperimentsResponse{}
	err := a.m.db.QueryProto("get_experiments", &resp.Experiments)
	if err != nil {
		return nil, err
	}
	var filtered []*experimentv1.Experiment
	for _, exp := range resp.Experiments {
		if filterExperiments(req, exp) {
			filtered = append(filtered, exp)
		}
	}
	resp.Experiments = filtered
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
