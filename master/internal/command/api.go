package command

import (
	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
)

const (
	// CommandActorPath is the path of the actor that manages commands.
	CommandActorPath = "commands"
	// NotebookActorPath is the path of the actor that manages notebooks.
	NotebookActorPath = "notebooks"
	// ShellActorPath is the path of the actor that manages shells.
	ShellActorPath = "shells"
	// TensorboardActorPath is the path of the actor that manages tensorboards.
	TensorboardActorPath = "tensorboard"
)

// RegisterAPIHandler initializes and registers the API handlers for all command related features.
func RegisterAPIHandler(
	system *actor.System,
	echo *echo.Echo,
	db *db.PgDB,
	rm rm.ResourceManager,
	middleware ...echo.MiddlewareFunc,
) {
	commandManagerRef, _ := system.ActorOf(
		actor.Addr(CommandActorPath),
		&commandManager{db: db, rm: rm},
	)
	notebookManagerRef, _ := system.ActorOf(
		actor.Addr(NotebookActorPath),
		&notebookManager{db: db, rm: rm},
	)
	shellManagerRef, _ := system.ActorOf(
		actor.Addr(ShellActorPath),
		&shellManager{db: db, rm: rm},
	)
	tensorboardManagerRef, _ := system.ActorOf(
		actor.Addr(TensorboardActorPath),
		&tensorboardManager{db: db, rm: rm},
	)

	// Wait for all managers to initialize.
	refs := []*actor.Ref{commandManagerRef, notebookManagerRef, shellManagerRef, tensorboardManagerRef}
	system.AskAll(actor.Ping{}, refs...).GetAll()

	if echo != nil {
		echo.Any("/commands*", api.Route(system, nil), middleware...)
		echo.Any("/notebooks*", api.Route(system, nil), middleware...)
		echo.Any("/shells*", api.Route(system, nil), middleware...)
		echo.Any("/tensorboard*", api.Route(system, nil), middleware...)
	}
}
