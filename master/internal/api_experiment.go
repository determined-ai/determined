package internal

import (
	"context"
	"strings"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetExperiments(
	_ context.Context, req *apiv1.GetExperimentsRequest) (*apiv1.GetExperimentsResponse, error) {
	resp := &apiv1.GetExperimentsResponse{}
	if err := a.m.db.QueryProto("get_experiments", &resp.Experiments); err != nil {
		return nil, err
	}
	a.filter(&resp.Experiments, func(i int) bool {
		v := resp.Experiments[i]
		return filterAll(
			func() bool { return req.Archived == nil || req.Archived.Value == v.Archived },
			func() bool {
				return strings.Contains(
					strings.ToLower(v.Description), strings.ToLower(req.Description))
			},
			checkIn(v.State, req.States),
			checkIn(v.Username, req.Users),
		)
	})
	a.sort(resp.Experiments, req.OrderBy, req.SortBy, apiv1.GetExperimentsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Experiments, req.Offset, req.Limit)
}
