package command

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
)

// RegisterAPIHandler initializes and registers the API handlers for all command related features.
func RegisterAPIHandler(
	system *actor.System,
	echo *echo.Echo,
	db *db.PgDB,
	proxyRef *actor.Ref,
	timeout int,
	middleware ...echo.MiddlewareFunc,
) {
	system.ActorOf(actor.Addr("commands"), &commandManager{
		db: db,
	})
	echo.Any("/commands*", api.Route(system, nil), middleware...)

	system.ActorOf(actor.Addr("notebooks"), &notebookManager{
		db: db,
	})
	echo.Any("/notebooks*", api.Route(system, nil), middleware...)

	system.ActorOf(actor.Addr("shells"), &shellManager{
		db: db,
	})
	echo.Any("/shells*", api.Route(system, nil), middleware...)

	system.ActorOf(actor.Addr("tensorboard"), &tensorboardManager{
		db:       db,
		proxyRef: proxyRef,
		timeout:  time.Duration(timeout) * time.Second,
	})
	echo.Any("/tensorboard*", api.Route(system, nil), middleware...)
}
