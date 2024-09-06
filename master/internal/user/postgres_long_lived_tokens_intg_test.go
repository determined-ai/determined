//go:build integration
// +build integration

package user

import (
	"context"
	"testing"

	"github.com/o1egl/paseto"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TestDeleteAndCreateLongLivedToken tests deleting and creating token with default lifespan.
func TestDeleteAndCreateLongLivedToken(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a LongLivedToken.
	token, err := DeleteAndCreateLongLivedToken(context.TODO(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, token)

	restoredToken := restoreTokenInfo(token, t)

	expLifespan := DefaultTokenLifespan
	actLifespan := restoredToken.ExpiresAt.Sub(restoredToken.CreatedAt)
	require.Equal(t, expLifespan, actLifespan)

	tokenInfo, err := GetLongLivedTokenInfo(context.TODO(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, tokenInfo)
}

// TestDeleteAndCreateLongLivedTokenHasExpiresAt tests deleting and creating token with
// given lifespan.
func TestDeleteAndCreateLongLivedTokenHasExpiresAt(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a LongLivedToken with custom (Now() + 3 Months) Expiry Time.
	expLifespan := DefaultTokenLifespan * 3
	token, err := DeleteAndCreateLongLivedToken(context.TODO(), user.ID, WithTokenExpiresAt(&expLifespan))
	require.NoError(t, err)
	require.NotNil(t, token)

	restoredToken := restoreTokenInfo(token, t)

	actLifespan := restoredToken.ExpiresAt.Sub(restoredToken.CreatedAt)
	require.Equal(t, expLifespan, actLifespan)

	tokenInfo, err := GetLongLivedTokenInfo(context.TODO(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, tokenInfo)
}

// TestDeleteLongLivenTokenByUserID tests deleting token info for given userId.
func TestDeleteLongLivenTokenByUserID(t *testing.T) {
	userID, _, _, err := addTestSession()
	require.NoError(t, err)

	token, err := DeleteAndCreateLongLivedToken(context.TODO(), userID)
	require.NoError(t, err)
	require.NotNil(t, token)

	err = DeleteLongLivenTokenByUserID(context.TODO(), userID)
	require.NoError(t, err)

	_, err = GetLongLivedTokenInfo(context.TODO(), userID)
	require.ErrorContains(t, err, "no rows found with user_id")
}

// TestDeleteLongLivenTokenByToken tests deleting token info for given token.
func TestDeleteLongLivenTokenByToken(t *testing.T) {
	userID, _, _, err := addTestSession()
	require.NoError(t, err)

	token, err := DeleteAndCreateLongLivedToken(context.TODO(), userID)
	require.NoError(t, err)
	require.NotNil(t, token)

	err = DeleteLongLivenTokenByToken(context.TODO(), token)
	require.NoError(t, err)

	_, err = GetLongLivedTokenInfo(context.TODO(), userID)
	require.ErrorContains(t, err, "no rows found with user_id")
}

// TestDeleteLongLivenTokenByTokenId tests deleting token info for given token id.
func TestDeleteLongLivenTokenByTokenId(t *testing.T) {
	userID, _, _, err := addTestSession()
	require.NoError(t, err)

	token, err := DeleteAndCreateLongLivedToken(context.TODO(), userID)
	require.NoError(t, err)
	require.NotNil(t, token)

	longLivedToken := restoreTokenInfo(token, t)

	err = DeleteLongLivedTokenByTokenID(context.TODO(), longLivedToken.ID)
	require.NoError(t, err)

	_, err = GetLongLivedTokenInfo(context.TODO(), userID)
	require.ErrorContains(t, err, "no rows found with user_id")
}

// TestGetLongLivenTokenInfoByUserID tests getting token info for given userId.
func TestGetLongLivenTokenInfoByUserID(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a LongLivedToken.
	token, err := DeleteAndCreateLongLivedToken(context.TODO(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, token)

	tokenInfo, err := GetLongLivedTokenInfo(context.TODO(), user.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, tokenInfo.UserID)
}

func restoreTokenInfo(token string, t *testing.T) model.LongLivedToken {
	var restoredToken model.LongLivedToken
	v2 := paseto.NewV2()
	err := v2.Verify(token, db.GetTokenKeys().PublicKey, &restoredToken, nil)
	require.NoError(t, err)

	return restoredToken
}
