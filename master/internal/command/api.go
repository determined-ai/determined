package command

import (
	"time"

	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// RegisterAPIHandler initializes and registers the API handlers for all command related features.
func RegisterAPIHandler(
	system *actor.System,
	echo *echo.Echo,
	db *db.PgDB,
	proxyRef *actor.Ref,
	timeout int,
	defaultAgentUserGroup model.AgentUserGroup,
	defaultTaskSpec *tasks.TaskSpec,
	middleware ...echo.MiddlewareFunc,
) {
	system.ActorOf(actor.Addr("commands"), &commandManager{
		defaultAgentUserGroup: defaultAgentUserGroup,
		db:                    db,
		defaultTaskSpec:       defaultTaskSpec,
	})
	echo.Any("/commands*", api.Route(system, nil), middleware...)

	system.ActorOf(actor.Addr("notebooks"), &notebookManager{
		defaultAgentUserGroup: defaultAgentUserGroup,
		db:                    db,
		defaultTaskSpec:       defaultTaskSpec,
	})
	echo.Any("/notebooks*", api.Route(system, nil), middleware...)

	system.ActorOf(actor.Addr("shells"), &shellManager{
		defaultAgentUserGroup: defaultAgentUserGroup,
		db:                    db,
		defaultTaskSpec:       defaultTaskSpec,
	})
	echo.Any("/shells*", api.Route(system, nil), middleware...)

	system.ActorOf(actor.Addr("tensorboard"), &tensorboardManager{
		defaultAgentUserGroup: defaultAgentUserGroup,
		db:                    db,
		defaultTaskSpec:       defaultTaskSpec,
		proxyRef:              proxyRef,
		timeout:               time.Duration(timeout) * time.Second,
	})
	echo.Any("/tensorboard*", api.Route(system, nil), middleware...)
}
