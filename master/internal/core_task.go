package internal

import (
	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/context"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

func (m *Master) getTasks(c echo.Context) (interface{}, error) {
	summary, err := m.rm.GetAllocationSummaries(m.system, sproto.GetAllocationSummaries{})
	if err != nil {
		return nil, err
	}

	curUser := c.(*context.DetContext).MustGetUser()
	for k, s := range summary {
		t, err := m.db.TaskByID(s.TaskID)
		if err != nil {
			return nil, err
		}

		switch t.TaskType {
		case model.TaskTypeTrial:
			exp, err := m.db.ExperimentWithoutConfigByTaskID(t.TaskID)
			if err != nil {
				return nil, err
			}

			if ok, err := expauth.AuthZProvider.Get().CanGetExperiment(curUser, exp); err != nil {
				return nil, err
			} else if !ok {
				delete(summary, k)
			}
		default:
			continue // TODO(nick) add AuthZ for other task types.
		}
	}
	return summary, nil
}
