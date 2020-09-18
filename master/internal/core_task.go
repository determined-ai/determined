package internal

import (
	"net/http"

	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/scheduler"
)

func (m *Master) getTasks(c echo.Context) (interface{}, error) {
	return m.system.Ask(m.rp, scheduler.GetTaskSummaries{}).Get(), nil
}

func (m *Master) getTask(c echo.Context) (interface{}, error) {
	args := struct {
		TaskID string `path:"task_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	id := scheduler.TaskID(args.TaskID)
	resp := m.system.Ask(m.rp, scheduler.GetTaskSummary{ID: &id})
	if resp.Empty() {
		return nil, echo.NewHTTPError(http.StatusNotFound, "task not found: %s", args.TaskID)
	}
	return resp.Get(), nil
}
