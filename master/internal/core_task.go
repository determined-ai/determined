package internal

import (
	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/context"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/sproto"
)

func (m *Master) getTasks(c echo.Context) (interface{}, error) {
	summary, err := m.rm.GetAllocationSummaries(sproto.GetAllocationSummaries{})
	if err != nil {
		return nil, err
	}

	curUser := c.(*context.DetContext).MustGetUser()
	ctx := c.Request().Context()
	for allocationID, allocationSummary := range summary {
		isExp, exp, err := expFromTaskID(ctx, allocationSummary.TaskID)
		if err != nil {
			return nil, err
		}

		if !isExp {
			_, err = canAccessNTSCTask(ctx, curUser, summary[allocationID].TaskID)
		} else {
			err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, curUser, exp)
		}
		if authz.IsPermissionDenied(err) {
			delete(summary, allocationID)
		} else if err != nil {
			return nil, err
		}
	}
	return summary, nil
}
