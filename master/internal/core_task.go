package internal

import (
	"github.com/labstack/echo/v4"

	//"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/sproto"
	//"github.com/determined-ai/determined/master/pkg/model"
)

// TODO now
func (m *Master) getTasks(c echo.Context) (interface{}, error) {
	return m.rm.GetAllocationSummaries(m.system, sproto.GetAllocationSummaries{})
}
