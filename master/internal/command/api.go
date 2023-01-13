package command

import (
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
)

// ErrAPIRemoved is an error to inform the client they are calling an old, removed API.
var ErrAPIRemoved = errors.New(`the API being called was removed,
please ensure the client consuming the API is up to date and report a bug if the problem persists`)

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
	taskLogger *task.Logger,
	middleware ...echo.MiddlewareFunc,
) {
	commandManagerRef, _ := system.ActorOf(
		actor.Addr(CommandActorPath),
		&commandManager{db: db, rm: rm, taskLogger: taskLogger},
	)
	notebookManagerRef, _ := system.ActorOf(
		actor.Addr(NotebookActorPath),
		&notebookManager{db: db, rm: rm, taskLogger: taskLogger},
	)
	shellManagerRef, _ := system.ActorOf(
		actor.Addr(ShellActorPath),
		&shellManager{db: db, rm: rm, taskLogger: taskLogger},
	)
	tensorboardManagerRef, _ := system.ActorOf(
		actor.Addr(TensorboardActorPath),
		&tensorboardManager{db: db, rm: rm, taskLogger: taskLogger},
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
