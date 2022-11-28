package command

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// CommandAuthZBasic is basic OSS controls.
type CommandAuthZBasic struct{}

// CanGetCommand always returns true and a nill error.
func (a *CommandAuthZBasic) CanGetCommand(
	ctx context.Context, curUser model.User, c *command,
) (canGetCmd bool, serverError error) {
	return true, nil
}

// CanGetCommandArtifacts always returns a nil error.
func (a *CommandAuthZBasic) CanGetCommandArtifacts(
	ctx context.Context, curUser model.User, c *command,
) error {
	return nil
}

// CanDeleteCommand returns an error if the command
// is not owned by the current user and the current user is not an admin.
func (a *CommandAuthZBasic) CanDeleteCommand(
	ctx context.Context, curUser model.User, c *command,
) error {
	if !curUser.Admin {
		return fmt.Errorf("non admin users may not delete other user's commands")
	}
	return nil
}

// FilterCommandsQuery returns the query unmodified and a nil error.
func (a *CommandAuthZBasic) FilterCommandsQuery(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	return query, nil
}

// CanEditCommand always returns a nil error.
func (a *CommandAuthZBasic) CanEditCommand(
	ctx context.Context, curUser model.User, c *command,
) error {
	return nil
}

// CanEditCommandsMetadata always returns a nil error.
func (a *CommandAuthZBasic) CanEditCommandsMetadata(
	ctx context.Context, curUser model.User, c *command,
) error {
	return nil
}

// CanCreateCommand always returns a nil error.
func (a *CommandAuthZBasic) CanCreateCommand(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace, c *command,
) error {
	return nil
}

// CanSetCommandsMaxSlots always returns a nil error.
func (a *CommandAuthZBasic) CanSetCommandsMaxSlots(
	ctx context.Context, curUser model.User, c *command, slots int,
) error {
	return nil
}

// CanSetCommandsWeight always returns a nil error.
func (a *CommandAuthZBasic) CanSetCommandsWeight(
	ctx context.Context, curUser model.User, c *command, weight float64,
) error {
	return nil
}

// CanSetCommandsPriority always returns a nil error.
func (a *CommandAuthZBasic) CanSetCommandsPriority(
	ctx context.Context, curUser model.User, c *command, priority int,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &CommandAuthZBasic{})
}
