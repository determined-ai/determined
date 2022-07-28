//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/metadata"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

var (
	pgDB      *db.PgDB
	authzUser *mocks.UserAuthZ = &mocks.UserAuthZ{}
)

func setup_user_authz(t *testing.T) (*apiServer, *mocks.UserAuthZ, context.Context) {
	if pgDB == nil {
		pgDB = db.MustResolveTestPostgres(t)
		db.MustMigrateTestPostgres(t, pgDB, "file://../static/migrations")
		require.NoError(t, etc.SetRootPath("../static/srv"))

		user.AuthZProvider.Register("mock", authzUser)
		config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}
	}

	api := &apiServer{m: &Master{
		db: pgDB,
		config: &config.Config{
			InternalConfig: config.InternalConfig{},
		},
	}}

	resp, err := api.Login(context.TODO(), &apiv1.LoginRequest{Username: "determined"})
	require.NoError(t, err, "Couldn't login")
	ctx := metadata.NewIncomingContext(context.TODO(),
		metadata.Pairs("x-user-token", fmt.Sprintf("Bearer %s", resp.Token)))

	return api, authzUser, ctx
}

func TestAuthzGetUsers(t *testing.T) {
	api, authzUsers, ctx := setup_user_authz(t)

	// Error just passes error through.
	expectedErr := fmt.Errorf("filterUseList")
	authzUsers.On("FilterUserList", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()
	_, err := api.GetUsers(ctx, &apiv1.GetUsersRequest{})
	require.Equal(t, expectedErr, err)

	// Nil error returns whatever FilterUserList returns.
	users := []model.FullUser{
		{Username: "a"},
		{Username: "b"},
	}
	authzUsers.On("FilterUserList", mock.Anything, mock.Anything).Return(users, nil).Once()
	actual, err := api.GetUsers(ctx, &apiv1.GetUsersRequest{})
	require.NoError(t, err)

	var expected apiv1.GetUsersResponse
	for _, u := range users {
		expected.Users = append(expected.Users, toProtoUserFromFullUser(u))
	}
	require.Equal(t, actual.Users, expected.Users)
}
