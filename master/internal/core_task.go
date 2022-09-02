package internal

import (
	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TODO auth
func (m *Master) getTasks(c echo.Context) (interface{}, error) {
	return m.rm.GetAllocationSummaries(m.system, sproto.GetAllocationSummaries{})
}

// TODO auth
func (m *Master) getTask(c echo.Context) (interface{}, error) {
	args := struct {
		AllocationID string `path:"allocation_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	return m.rm.GetAllocationSummary(m.system, sproto.GetAllocationSummary{
		ID: model.AllocationID(args.AllocationID),
	})
}
