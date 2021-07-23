package internal

import (
	"fmt"
	"net/http"

	"github.com/determined-ai/determined/master/internal/db"

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

	eID, rID, err := m.db.TrialExperimentAndRequestID(args.TrialID)
	switch {
	case errors.Is(err, db.ErrNotFound):
		return nil, trialNotFound
	case err != nil:
		return nil, err
	}
	trialAddr := actor.Addr("experiments", eID, rID)

	resp := m.system.AskAt(trialAddr, model.StoppingCanceledState)
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
