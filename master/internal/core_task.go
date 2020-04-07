package internal

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/scheduler"
)

func (m *Master) getTasks(c echo.Context) (interface{}, error) {
	return m.system.Ask(m.cluster, scheduler.GetTaskSummaries{}).Get(), nil
}

func (m *Master) getTask(c echo.Context) (interface{}, error) {
	args := struct {
		TaskID string `path:"task_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	id := scheduler.TaskID(args.TaskID)
	resp := m.system.Ask(m.cluster, scheduler.GetTaskSummary{ID: &id})
	if resp.Empty() {
		return nil, echo.NewHTTPError(http.StatusNotFound, "task not found: %s", args.TaskID)
	}
	return resp.Get(), nil
}

func (m *Master) deleteTask(c echo.Context) (interface{}, error) {
	args := struct {
		TaskID string `path:"task_id"`
		Force  *bool  `query:"force"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	if args.Force == nil {
		force := false
		args.Force = &force
	}
	killedTask := m.system.Ask(m.cluster, scheduler.TerminateTask{
		TaskID: scheduler.TaskID(args.TaskID), Forcible: *args.Force,
	}).Get().(*scheduler.Task)
	if killedTask == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("task not found: %s", args.TaskID))
	}
	return nil, nil
}
