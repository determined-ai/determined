package command

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// CommandAuthZBasic is basic OSS controls.
type CommandAuthZBasic struct{}

// CanGetCommand always returns true and a nill error.
func (a *CommandAuthZBasic) CanGetCommand(
	ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec,
) (canGetCmd bool, serverError error) {
	return true, nil
}

// CanGetCommandArtifacts always returns a nil error.
func (a *CommandAuthZBasic) CanGetCommandArtifacts(
	ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec,
) error {
	return nil
}

// FilterCommandsQuery returns the query unmodified and a nil error.
func (a *CommandAuthZBasic) FilterCommandsQuery(
	ctx context.Context, curUser model.User, workspace *model.Workspace, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	return query, nil
}

// CanEditCommand always returns a nil error.
func (a *CommandAuthZBasic) CanEditCommand(
	ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec,
) error {
	return nil
}

// CanEditCommandsMetadata always returns a nil error.
func (a *CommandAuthZBasic) CanEditCommandsMetadata(
	ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec,
) error {
	return nil
}

// CanCreateCommand always returns a nil error.
func (a *CommandAuthZBasic) CanCreateCommand(
	ctx context.Context, curUser model.User, workspace *model.Workspace, c *tasks.GenericCommandSpec,
) error {
	return nil
}

// CanSetCommandsMaxSlots always returns a nil error.
func (a *CommandAuthZBasic) CanSetCommandsMaxSlots(
	ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, slots int,
) error {
	return nil
}

// CanSetCommandsWeight always returns a nil error.
func (a *CommandAuthZBasic) CanSetCommandsWeight(
	ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, weight float64,
) error {
	return nil
}

// CanSetCommandsPriority always returns a nil error.
func (a *CommandAuthZBasic) CanSetCommandsPriority(
	ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, priority int,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &CommandAuthZBasic{})
}
