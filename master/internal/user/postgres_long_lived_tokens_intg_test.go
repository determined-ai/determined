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

	var restoredTokenInfo model.LongLivedToken
	v2 := paseto.NewV2()
	err = v2.Verify(token, db.GetTokenKeys().PublicKey, &restoredTokenInfo, nil)
	require.NoError(t, err)

	expLifespan := DefaultTokenLifespan
	actLifespan := restoredTokenInfo.ExpiresAt.Sub(restoredTokenInfo.CreatedAt)
	require.Equal(t, expLifespan, actLifespan)

	exists, err := getLongLivedTokensEntry(context.TODO(), user.ID)
	require.True(t, exists)
	require.NoError(t, err)
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

	var restoredTokenInfo model.LongLivedToken
	v2 := paseto.NewV2()
	err = v2.Verify(token, db.GetTokenKeys().PublicKey, &restoredTokenInfo, nil)
	require.NoError(t, err)

	actLifespan := restoredTokenInfo.ExpiresAt.Sub(restoredTokenInfo.CreatedAt)
	require.Equal(t, expLifespan, actLifespan)

	exists, err := getLongLivedTokensEntry(context.TODO(), user.ID)
	require.True(t, exists)
	require.NoError(t, err)
}

// TestDeleteLongLivenTokenByUserID tests deleting token info for given userId.
func TestDeleteLongLivenTokenByUserID(t *testing.T) {
	userID, _, _, err := addTestSession()
	require.NoError(t, err)

	tx, err := db.Bun().BeginTx(context.TODO(), nil)
	require.NoError(t, err)

	err = DeleteLongLivenTokenByUserID(context.TODO(), tx, userID)
	require.NoError(t, err)

	exists, err := getLongLivedTokensEntry(context.TODO(), userID)
	require.False(t, exists)
	require.NoError(t, err)
}

// TestDeleteLongLivenTokenByToken tests deleting token info for given token.
func TestDeleteLongLivenTokenByToken(t *testing.T) {
	userID, _, token, err := addTestSession()
	require.NoError(t, err)

	err = DeleteLongLivenTokenByToken(context.TODO(), token)
	require.NoError(t, err)

	exists, err := getLongLivedTokensEntry(context.TODO(), userID)
	require.False(t, exists)
	require.NoError(t, err)
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

func getLongLivedTokensEntry(ctx context.Context, userID model.UserID) (bool, error) {
	return db.Bun().NewSelect().Table("long_lived_tokens").
		Where("user_id = ?", userID).Exists(ctx)
}
