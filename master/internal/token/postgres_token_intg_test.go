//go:build integration
// +build integration

package token

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/o1egl/paseto"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

const desc = "test desc"

func TestMain(m *testing.M) {
	pgDB, _, err := db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	os.Exit(m.Run())
}

// TestCreateAccessToken tests revoking and creating token with default lifespan.
func TestCreateAccessToken(t *testing.T) {
	testUser, err := addTestUser(nil)
	require.NoError(t, err)

	// Add an access Token.
	token, tokenID, err := CreateAccessToken(context.TODO(), testUser.ID)
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotNil(t, tokenID)

	restoredToken := restoreTokenInfo(token, t)

	expLifespan := DefaultTokenLifespan
	actLifespan := restoredToken.Expiry.Sub(restoredToken.CreatedAt)
	require.Equal(t, expLifespan, actLifespan)

	tokenInfos, err := getAccessToken(context.TODO(), testUser.ID)
	require.NoError(t, err)
	require.NotNil(t, tokenInfos)

	// Loop through all the returned user sessions
	for _, tokenInfo := range tokenInfos {
		// For test cleanup: delete each session by ID
		err = user.DeleteSessionByID(context.TODO(), tokenInfo.ID)
		require.NoError(t, err)
	}
}

// TestCreateAccessTokenHasExpiry tests revoking and creating token with
// given lifespan and description.
func TestCreateAccessTokenHasExpiry(t *testing.T) {
	testUser, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a AccessToken with custom (Now() + 3 Months) Expiry Time.
	expLifespan := DefaultTokenLifespan * 3
	token, tokenID, err := CreateAccessToken(context.TODO(), testUser.ID,
		WithTokenExpiry(&expLifespan), WithTokenDescription(desc))
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotNil(t, tokenID)

	restoredToken := restoreTokenInfo(token, t)

	actLifespan := restoredToken.Expiry.Sub(restoredToken.CreatedAt)
	require.Equal(t, expLifespan.Truncate(time.Second), actLifespan.Truncate(time.Second))
	require.Equal(t, desc, restoredToken.Description.String)

	tokenInfos, err := getAccessToken(context.TODO(), testUser.ID)
	require.NoError(t, err)
	require.NotNil(t, tokenInfos)

	// Loop through all the returned user sessions
	for _, tokenInfo := range tokenInfos {
		// For test cleanup: delete each session by ID
		err = user.DeleteSessionByID(context.TODO(), tokenInfo.ID)
		require.NoError(t, err)
	}
}

// TestUpdateAccessToken tests the description and revocation status of the access token.
func TestUpdateAccessToken(t *testing.T) {
	userID, _, _, err := addTestSession()
	require.NoError(t, err)

	token, tokenID, err := CreateAccessToken(context.TODO(), userID)
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotNil(t, tokenID)

	accessToken := restoreTokenInfo(token, t)

	// Test before updating Access token
	description := "description"
	require.True(t, accessToken.RevokedAt.IsZero())
	require.NotEqual(t, description, accessToken.Description)

	opt := AccessTokenUpdateOptions{Description: &description, SetRevoked: true}
	tokenInfo, err := UpdateAccessToken(context.TODO(), model.TokenID(accessToken.ID), opt)
	require.NoError(t, err)

	// Test after updating access token
	require.False(t, tokenInfo.RevokedAt.IsZero())
	require.Contains(t, description, tokenInfo.Description.String)

	// Delete from DB by UserID for cleanup
	err = user.DeleteSessionByID(context.TODO(), tokenInfo.ID)
	require.NoError(t, err)
}

// TestGetAccessTokenInfoByUserID tests getting access token info for given userId.
func TestGetAccessToken(t *testing.T) {
	testUser, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a AccessToken.
	token, tokenID, err := CreateAccessToken(context.TODO(), testUser.ID)
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotNil(t, tokenID)

	tokenInfos, err := getAccessToken(context.TODO(), testUser.ID)
	require.NoError(t, err)

	restoredTokeninfo := restoreTokenInfo(token, t)

	// Flag to check if userID is found in tokenInfos
	userIDFound := false
	restoredTokenIDFound := false

	// Loop through all the returned user sessions
	for _, tokenInfo := range tokenInfos {
		// Check if user ID matches
		if tokenInfo.UserID == testUser.ID {
			userIDFound = true
		}

		// Check if the token ID matches the restored token info
		if tokenInfo.ID == restoredTokeninfo.ID {
			restoredTokenIDFound = true
		}

		// For test cleanup: delete each session by ID
		err = user.DeleteSessionByID(context.TODO(), tokenInfo.ID)
		require.NoError(t, err)
	}

	// Assert that user.ID and restoredTokeninfo.ID are found in tokenInfos
	require.True(t, userIDFound, "User ID should be present in tokenInfos")
	require.True(t, restoredTokenIDFound, "Restored token ID should be present in tokenInfos")
}

func addTestUser(aug *model.AgentUserGroup, opts ...func(*model.User)) (*model.User, error) {
	testUser := model.User{Username: uuid.NewString()}
	for _, opt := range opts {
		opt(&testUser)
	}

	uid, err := user.Add(context.TODO(), &testUser, aug)
	if err != nil {
		return nil, fmt.Errorf("couldn't create new user: %w", err)
	}
	err = db.Bun().NewSelect().Table("users").Where("id = ?", uid).Scan(context.TODO(), &testUser)
	return &testUser, err
}

func addTestSession() (model.UserID, model.SessionID, string, error) {
	// Add a user.
	testUser, err := addTestUser(nil)
	if err != nil {
		return 0, 0, "", err
	}

	// Add a session.
	var session model.UserSession
	token, err := user.StartSession(context.TODO(), testUser)
	if err != nil {
		return 0, 0, "", fmt.Errorf("couldn't create new session: %w", err)
	}

	if err = db.Bun().NewSelect().Table("user_sessions").
		Where("user_id = ?", testUser.ID).
		Where("token_type = ?", model.TokenTypeUserSession).
		Scan(context.TODO(), &session); err != nil {
		return 0, 0, "", fmt.Errorf("couldn't create new session: %w", err)
	}
	return testUser.ID, session.ID, token, nil
}

func restoreTokenInfo(token string, t *testing.T) model.UserSession {
	var restoredToken model.UserSession
	v2 := paseto.NewV2()
	err := v2.Verify(token, db.GetTokenKeys().PublicKey, &restoredToken, nil)
	require.NoError(t, err)

	return restoredToken
}

func getAccessToken(ctx context.Context, userID model.UserID) ([]model.UserSession, error) {
	var tokenInfos []model.UserSession // To store the token info for the given user_id

	// Execute the query to fetch the active token info for the given user_id
	err := db.Bun().NewSelect().
		Table("user_sessions").
		Where("user_id = ?", userID).
		Where("revoked_at IS NULL").
		Where("token_type = ?", model.TokenTypeAccessToken).
		Scan(ctx, &tokenInfos)
	if err != nil {
		return nil, err
	}
	return tokenInfos, nil
}
