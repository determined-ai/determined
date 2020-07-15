package command

import (
	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/model"
)

// RegisterAPIHandler initializes and registers the API handlers for all command related features.
func RegisterAPIHandler(
	system *actor.System,
	echo *echo.Echo,
	db *db.PgDB,
	cID string,
	defaultAgentUserGroup model.AgentUserGroup,
	middleware ...echo.MiddlewareFunc,
) {
	system.ActorOf(actor.Addr("commands"), &commandManager{
		defaultAgentUserGroup: defaultAgentUserGroup,
		db:                    db,
		clusterID:             cID,
	})
	echo.Any("/commands*", api.Route(system, nil), middleware...)

	system.ActorOf(actor.Addr("notebooks"), &notebookManager{
		defaultAgentUserGroup: defaultAgentUserGroup,
		db:                    db,
		clusterID:             cID,
	})
	echo.Any("/notebooks*", api.Route(system, nil), middleware...)

	system.ActorOf(actor.Addr("shells"), &shellManager{
		defaultAgentUserGroup: defaultAgentUserGroup,
		db:                    db,
		clusterID:             cID,
	})
	echo.Any("/shells*", api.Route(system, nil), middleware...)

	system.ActorOf(actor.Addr("tensorboard"), &tensorboardManager{
		defaultAgentUserGroup: defaultAgentUserGroup,
		db:                    db,
		clusterID:             cID,
	})
	echo.Any("/tensorboard*", api.Route(system, nil), middleware...)
}
