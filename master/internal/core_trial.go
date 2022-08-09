package internal

import (
	"github.com/labstack/echo/v4"
)

// TODO(ilia): These APIs are deprecated and will be removed in a future release.
func (m *Master) getTrial(c echo.Context) (interface{}, error) {
	return m.db.RawQuery("get_trial", c.Param("trial_id"))
}

func (m *Master) getTrialMetrics(c echo.Context) (interface{}, error) {
	return m.db.RawQuery("get_trial_metrics", c.Param("trial_id"))
}
