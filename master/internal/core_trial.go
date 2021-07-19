package internal

import (
	"fmt"
	"net/http"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/actor"
)

func (m *Master) postTrialKill(c echo.Context) (interface{}, error) {
	args := struct {
		TrialID int `path:"trial_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}

	trial, err := m.db.TrialByID(args.TrialID)
	if err != nil {
		return nil, err
	}
	resp := m.system.AskAt(actor.Addr("experiments", trial.ExperimentID),
		getTrial{trialID: args.TrialID})
	if resp.Source() == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("active experiment not found: %d", trial.ExperimentID))
	}
	if resp.Empty() {
		return nil, echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("active trial not found: %d", args.TrialID))
	}
	resp = m.system.AskAt(resp.Get().(*actor.Ref).Address(), model.StoppingCanceledState)
	if resp.Source() == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("active trial not found: %d", args.TrialID))
	}
	if _, notTimedOut := resp.GetOrTimeout(defaultAskTimeout); !notTimedOut {
		return nil, errors.Errorf("attempt to kill trial timed out")
	}
	return nil, nil
}

func (m *Master) getTrial(c echo.Context) (interface{}, error) {
	return m.db.RawQuery("get_trial", c.Param("trial_id"))
}

func (m *Master) getTrialDetails(c echo.Context) (interface{}, error) {
	args := struct {
		TrialID int `path:"trial_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	return m.db.TrialDetailsRaw(args.TrialID)
}

func (m *Master) getTrialMetrics(c echo.Context) (interface{}, error) {
	return m.db.RawQuery("get_trial_metrics", c.Param("trial_id"))
}
