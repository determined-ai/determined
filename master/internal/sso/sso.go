package sso

import (
	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/aproto"
)

// AddProviderInfo uses the master config to add SSOProvider
// to the provided masterInfo. In OSS this just returns masterInfo.
func AddProviderInfo(config *config.Config, masterInfo aproto.MasterInfo) aproto.MasterInfo {
	return masterInfo
}

// InitializeNeededHandlers registers needed API handlers
// determined by master config. In OSS this is just a no-op.
func RegisterAPIHandlers(config *config.Config, db *db.PgDB, echo *echo.Echo) error {
	return nil
}
