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

func getAgentUserGroupFromUser(userID model.UserID) (*model.AgentUserGroup, error) {
	var aug model.AgentUserGroup
	err := db.Bun().NewSelect().Model(&aug).
		Relation("RelatedUser").
		Where("related_user.id = ?", userID).
		Scan(context.TODO())
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &aug, nil
}

type optionalAgentUserGroup struct {
	User *string
	UID  *int

	Group *string
	GID   *int
}

// TODO(ilia): Bun me.
func getAgentUserGroupFromExperiment(e *model.Experiment) (*optionalAgentUserGroup, error) {
	aug := optionalAgentUserGroup{}

	if e == nil {
		return &aug, nil
	}

	err := db.Bun().NewRaw(`
SELECT
	uid, user_ as user, gid, group_ as group
FROM workspaces JOIN projects ON workspaces.id = projects.workspace_id
WHERE projects.id = ?`,
		e.ProjectID).Scan(context.TODO(), &aug)
	return &aug, err
}

// GetAgentUserGroup returns AgentUserGroup for a user + (optional) experiment.
func GetAgentUserGroup(userID model.UserID, e *model.Experiment) (*model.AgentUserGroup, error) {
	expAug, err := getAgentUserGroupFromExperiment(e)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent user group from experiment: %w", err)
	}

	userAug, err := getAgentUserGroupFromUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent user group from user: %w", err)
	}

	if userAug == nil {
		userAug = &config.GetMasterConfig().Security.DefaultTask
	}

	// Merge exp AUG and user AUG.
	result := model.AgentUserGroup{
		UID:   userAug.UID,
		User:  userAug.User,
		GID:   userAug.GID,
		Group: userAug.Group,
	}
	if expAug.UID != nil {
		result.UID = *expAug.UID
	}
	if expAug.User != nil {
		result.User = *expAug.User
	}
	if expAug.GID != nil {
		result.GID = *expAug.GID
	}
	if expAug.Group != nil {
		result.Group = *expAug.Group
	}

	return &result, nil
}
