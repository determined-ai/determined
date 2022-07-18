package sso

import (
	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// AddProviderInfoToMasterResponse modifies passed in master response adds sso
// provider information. In OSS this is a no-op. While having two functions that
// just set a field is somewhat awkward it avoids having to have OSS have any
// requirements that masterResp/masterInfo has a field defined for provider info.
func AddProviderInfoToMasterResponse(config *config.Config, masterResp *apiv1.GetMasterResponse) {}

// AddProviderInfoToMasterInfo modifies passed in master info adds sso
// provider information. In OSS this is a no-op.
func AddProviderInfoToMasterInfo(config *config.Config, masterInfo *aproto.MasterInfo) {}

// RegisterAPIHandlers registers needed API handlers
// determined by master config. In OSS this is just a no-op.
func RegisterAPIHandlers(config *config.Config, db *db.PgDB, echo *echo.Echo) error {
	return nil
}
