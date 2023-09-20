package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

func getAgentUserGroupFromUser(
	ctx context.Context,
	userID model.UserID,
) (*model.AgentUserGroup, error) {
	var aug model.AgentUserGroup
	switch err := db.Bun().NewSelect().Table("agent_user_groups").
		Where("user_id = ?", userID).
		Scan(ctx, &aug); {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &aug, nil
	}
}

type optionalAgentUserGroup struct {
	User *string
	UID  *int

	Group *string
	GID   *int
}

func getAgentUserGroupFromWorkspaceID(
	ctx context.Context,
	workspaceID int,
) (*optionalAgentUserGroup, error) {
	var aug optionalAgentUserGroup
	err := db.Bun().NewSelect().Table("workspaces").
		ColumnExpr("uid, user_ AS user, gid, group_ AS group").
		Where("id = ?", workspaceID).Scan(ctx, &aug)

	return &aug, err
}

// GetAgentUserGroup returns AgentUserGroup for a user + (optional) workspace.
func GetAgentUserGroup(
	ctx context.Context,
	userID model.UserID,
	workspaceID int,
) (*model.AgentUserGroup, error) {
	workspaceAug, err := getAgentUserGroupFromWorkspaceID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent user group from experiment: %w", err)
	}

	userAug, err := getAgentUserGroupFromUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent user group from user: %w", err)
	}

	if userAug == nil {
		userAug = &config.GetMasterConfig().Security.DefaultTask
	}

	// Merge workspace AUG and user AUG.
	result := model.AgentUserGroup{
		UID:   userAug.UID,
		User:  userAug.User,
		GID:   userAug.GID,
		Group: userAug.Group,
	}
	if workspaceAug.UID != nil {
		result.UID = *workspaceAug.UID
	}
	if workspaceAug.User != nil {
		result.User = *workspaceAug.User
	}
	if workspaceAug.GID != nil {
		result.GID = *workspaceAug.GID
	}
	if workspaceAug.Group != nil {
		result.Group = *workspaceAug.Group
	}

	return &result, nil
}
