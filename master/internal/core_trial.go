package internal

import (
	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/api"
)

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
