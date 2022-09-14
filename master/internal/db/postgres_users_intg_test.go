//go:build integration
// +build integration

package db

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestUserGroupCreation(t *testing.T) {
	etc.SetRootPath(RootFromDB)
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	user := &model.User{Username: uuid.New().String()}
	agentUserGroup := &model.AgentUserGroup{
		UID:   1023,
		User:  "linuxuser",
		GID:   1034,
		Group: "linuxgroup",
	}
	userID, err := db.AddUser(user, agentUserGroup)
	require.NoError(t, err)
	agentUserGroup.UserID = userID

	// User agent group was created?
	actualAgentUserGroup, err := db.AgentUserGroup(userID)
	require.NoError(t, err)
	require.Equal(t, agentUserGroup, actualAgentUserGroup)

	// User personal group was created?
	groups, counts, _, err := SearchGroup(ctx, "", userID, 0, 0)
	require.NoError(t, err)
}
