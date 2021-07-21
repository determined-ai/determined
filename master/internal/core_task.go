package internal

import (
	"net/http"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/sproto"
)

func (m *Master) getTasks(c echo.Context) (interface{}, error) {
	return m.system.Ask(m.rm, sproto.GetTaskSummaries{}).Get(), nil
}

func (m *Master) getTask(c echo.Context) (interface{}, error) {
	args := struct {
		TaskID string `path:"task_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	id := model.AllocationID(args.TaskID)
	resp := m.system.Ask(m.rm, sproto.GetTaskSummary{ID: &id})
	if resp.Empty() {
		return nil, echo.NewHTTPError(http.StatusNotFound, "task not found: %s", args.TaskID)
	}
	return resp.Get(), nil
}
