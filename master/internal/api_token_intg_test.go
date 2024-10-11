//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/token"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

var authzToken *mocks.TokenAuthZ

const (
	lifespan = "5s"
	desc     = "test desc"
)

// TestPostAccessToken tests given user's WITHOUT lifespan input
// POST /api/v1/users/{user_Id}/token - Create and get a user's access token.
func TestPostAccessToken(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	userID, err := getTestUser(ctx)
	require.NoError(t, err)

	// Without lifespan input
	resp, err := api.PostAccessToken(ctx, &apiv1.PostAccessTokenRequest{
		UserId: int32(userID),
	})
	token, tokenID := resp.Token, resp.TokenId
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotNil(t, tokenID)

	err = checkOutput(ctx, t, api, userID, "", "")
	require.NoError(t, err)

	// cleaning test data
	err = user.DeleteSessionByID(context.TODO(), model.SessionID(userID))
	require.NoError(t, err)
}

// TestPostAccessTokenWithLifespan tests given user's  WITH lifespan, description input
// POST /api/v1/users/{user_Id}/token - Create and get a user's access token
// Input body contains lifespan = "5s or "2h".
func TestPostAccessTokenWithLifespan(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	userID, err := getTestUser(ctx)
	require.NoError(t, err)

	// With lifespan input
	resp, err := api.PostAccessToken(ctx, &apiv1.PostAccessTokenRequest{
		UserId:      int32(userID),
		Lifespan:    lifespan,
		Description: desc,
	})
	token, tokenID := resp.Token, resp.TokenId
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotNil(t, tokenID)

	err = checkOutput(ctx, t, api, userID, lifespan, desc)
	require.NoError(t, err)

	// cleaning test data
	err = user.DeleteSessionByID(context.TODO(), model.SessionID(userID))
	require.NoError(t, err)
}

// TestGetAccessTokens tests all access token info
// GET /api/v1/users/tokens - Get all access token info
// from user_sessions db for admin.
func TestGetAccessTokens(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	// Create test user 1 and do not revoke or set description
	userID1, err := getTestUser(ctx)
	require.NoError(t, err)

	createTestToken(ctx, t, api, userID1)

	usernameForGivenUserID, err := getUsernameForGivenUserID(ctx, userID1)
	require.NoError(t, err)
	filter := fmt.Sprintf(`{"username":"%s"}`, usernameForGivenUserID)

	tokenInfo1, err := api.GetAccessTokens(ctx, &apiv1.GetAccessTokensRequest{
		Filter: filter,
	})
	require.NoError(t, err)
	require.NotNil(t, tokenInfo1)
	// Loop through the returned access token records
	userIDFound := false
	tokenID1 := 0
	for _, tokenInfo := range tokenInfo1.TokenInfo {
		// Check if user ID matches
		if tokenInfo.UserId == int32(userID1) {
			userIDFound = true
			tokenID1 = int(tokenInfo.Id)
		}
	}
	require.True(t, userIDFound, "User ID should be present in tokenInfo1")

	// Create test user 2 and revoke and set description
	userID2, err := getTestUser(ctx)
	require.NoError(t, err)

	createTestToken(ctx, t, api, userID2)

	usernameForGivenUserID, err = getUsernameForGivenUserID(ctx, userID2)
	require.NoError(t, err)
	filter = fmt.Sprintf(`{"username":"%s"}`, usernameForGivenUserID)

	// Tests TestGetAccessToken info for giver userID
	tokenInfo2, err := api.GetAccessTokens(ctx, &apiv1.GetAccessTokensRequest{
		Filter: filter,
	})
	require.NoError(t, err)
	require.NotNil(t, tokenInfo2)
	// Loop through the returned access token records
	userIDFound = false
	tokenID2 := 0
	for _, tokenInfo := range tokenInfo2.TokenInfo {
		// Check if user ID matches
		if tokenInfo.UserId == int32(userID2) {
			userIDFound = true
			tokenID2 = int(tokenInfo.Id)
		}
	}
	require.True(t, userIDFound, "User ID should be present in tokenInfo2")

	description := "test desc"
	// Tests TestPatchAccessToken info for giver tokenID
	_, err = api.PatchAccessToken(ctx, &apiv1.PatchAccessTokenRequest{
		TokenId:     int32(tokenID2),
		Description: &description,
		SetRevoked:  true,
	})
	require.NoError(t, err)

	resp, err := api.GetAccessTokens(ctx, &apiv1.GetAccessTokensRequest{})
	require.NoError(t, err)

	for _, u := range resp.TokenInfo {
		if model.UserID(u.Id) == model.UserID(tokenID1) {
			require.False(t, u.Revoked)
			require.NotEqual(t, description, u.Description)
		} else if model.UserID(u.Id) == model.UserID(tokenID2) {
			require.True(t, u.Revoked)
			require.Equal(t, description, u.Description)
		}
	}

	// Clean up of test users
	for _, u := range resp.TokenInfo {
		err = user.DeleteSessionByID(context.TODO(), model.SessionID(u.Id))
		require.NoError(t, err)
	}
}

// TestAuthzOtherAccessToken tests authorization of user creating/viewing/patching
// given user's token.
func TestAuthzOtherAccessToken(t *testing.T) {
	api, authzToken, curUser, ctx := setupTokenAuthzTest(t, nil)

	// POST API Auth check
	expectedErr := status.Error(codes.PermissionDenied, "canCreateAccessToken")
	authzToken.On("CanCreateAccessToken", mock.Anything, curUser, curUser).
		Return(fmt.Errorf("canCreateAccessToken")).Once()

	_, err := api.PostAccessToken(ctx, &apiv1.PostAccessTokenRequest{
		UserId: int32(curUser.ID),
	})
	require.Equal(t, expectedErr.Error(), err.Error())

	// GET API Auth check
	var query bun.SelectQuery
	expectedErr = status.Error(codes.PermissionDenied, "canGetAccessTokens")
	authzToken.On("CanGetAccessTokens", mock.Anything, curUser, mock.Anything, curUser.ID).
		Return(&query, fmt.Errorf("canGetAccessTokens")).Once()

	filter := fmt.Sprintf(`{"username":"%s"}`, curUser.Username)
	_, err = api.GetAccessTokens(ctx, &apiv1.GetAccessTokensRequest{
		Filter: filter,
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func checkOutput(ctx context.Context, t *testing.T, api *apiServer, userID model.UserID,
	lifespan string, desc string,
) error {
	usernameForGivenUserID, err := getUsernameForGivenUserID(ctx, userID)
	require.NoError(t, err)

	filter := fmt.Sprintf(`{"username":"%s"}`, usernameForGivenUserID)
	tokenInfos, err := api.GetAccessTokens(ctx, &apiv1.GetAccessTokensRequest{
		Filter: filter,
	})
	require.NoError(t, err)
	require.NotNil(t, tokenInfos)

	tokenID := model.TokenID(0)
	if desc != "" {
		descFound := false
		for _, tokenInfo := range tokenInfos.TokenInfo {
			// Check if user ID matches
			if tokenInfo.Description == desc {
				descFound = true
				tokenID = model.TokenID(tokenInfo.Id)
			}
		}
		require.True(t, descFound, "Desc should be present in tokenInfo")
	}

	if lifespan != "" {
		err = testSetLifespan(ctx, t, userID, lifespan, tokenID)
		require.NoError(t, err)
	}

	return nil
}

func createTestToken(ctx context.Context, t *testing.T, api *apiServer, userID model.UserID) {
	if userID == 0 {
		// Create a test token for current user without lifespan input
		resp, err := api.PostAccessToken(ctx, &apiv1.PostAccessTokenRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp.Token)
		require.NotNil(t, resp.TokenId)
	} else {
		// Create a test token for user_id without lifespan input
		resp, err := api.PostAccessToken(ctx, &apiv1.PostAccessTokenRequest{
			UserId: int32(userID),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Token)
		require.NotNil(t, resp.TokenId)
	}
}

func getTestUser(ctx context.Context) (model.UserID, error) {
	return user.Add(
		ctx,
		&model.User{
			Username: uuid.New().String(),
			Remote:   false,
		},
		nil,
	)
}

func testSetLifespan(ctx context.Context, t *testing.T, userID model.UserID, lifespan string,
	tokenID model.TokenID,
) error {
	expLifespan := token.DefaultTokenLifespan
	var err error
	if lifespan != "" {
		expLifespan, err = time.ParseDuration(lifespan)
		if err != nil {
			return fmt.Errorf("Invalid duration format")
		}
	}
	var expiry, createdAt time.Time
	err = db.Bun().NewSelect().
		Table("user_sessions").
		Column("expiry", "created_at").
		Where("user_id = ?", userID).
		Where("token_type = ?", model.TokenTypeAccessToken).
		Where("id = ?", tokenID).
		Scan(ctx, &expiry, &createdAt)
	if err != nil {
		return fmt.Errorf("Error getting the set lifespan, creation time")
	}

	actLifespan := expiry.Sub(createdAt)
	require.Equal(t, expLifespan, actLifespan)

	return nil
}

func getUsernameForGivenUserID(ctx context.Context, userID model.UserID) (string, error) {
	var usernameForGivenUserID string
	err := db.Bun().NewSelect().
		Table("users").
		Column("username").
		Where("id = ?", userID).
		Scan(ctx, &usernameForGivenUserID)
	if err != nil {
		return "", fmt.Errorf("Error getting username for the user")
	}
	return usernameForGivenUserID, nil
}

// pgdb can be nil to use the singleton database for testing.
func setupTokenAuthzTest(
	t *testing.T, pgdb *db.PgDB,
	altMockRM ...*mocks.ResourceManager,
) (*apiServer, *mocks.TokenAuthZ, model.User, context.Context) {
	api, curUser, ctx := setupAPITest(t, pgdb, altMockRM...)

	if authzToken == nil {
		authzToken = &mocks.TokenAuthZ{}
		token.AuthZProvider.Register("mock", authzToken)
		config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}
	}
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}

	return api, authzToken, curUser, ctx
}
