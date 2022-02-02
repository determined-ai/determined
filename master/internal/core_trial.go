package internal

import (
	"github.com/labstack/echo/v4"
)

func (m *Master) getTrial(c echo.Context) (interface{}, error) {
	return m.db.RawQuery("get_trial", c.Param("trial_id"))
}
