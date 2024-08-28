//go:build integration
// +build integration

package user

import (
	"context"
	"testing"
	"time"

	"github.com/o1egl/paseto"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestCreateLongLivedToken(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a LongLivedToken.
	token, err := CreateLongLivedToken(context.TODO(), user)
	require.NoError(t, err)
	require.NotNil(t, token)

	exists, err := db.Bun().NewSelect().Table("long_lived_tokens").
		Where("user_id = ?", user.ID).Exists(context.TODO())
	require.True(t, exists)
	require.NoError(t, err)
}

func TestCreateLongLivedTokenHasExpiryTime(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a LongLivedToken with custom (Now() + 3 Months) Expiry Time.
	expiryTime := time.Now().Add(TokenExpirationDuration * 3) // Now() + 3 Months
	token, err := CreateLongLivedToken(context.TODO(), user, WithTokenExpiryTime(&expiryTime))
	require.NoError(t, err)
	require.NotNil(t, token)

	var restoredSession model.LongLivedToken
	v2 := paseto.NewV2()
	err = v2.Verify(token, db.GetTokenKeys().PublicKey, &restoredSession, nil)
	require.NoError(t, err)
	require.Equal(t, restoredSession.ExpiryTime, expiryTime)

	exists, err := db.Bun().NewSelect().Table("long_lived_tokens").
		Where("user_id = ?", user.ID).Exists(context.TODO())
	require.True(t, exists)
	require.NoError(t, err)
}

// func TestDeleteLongLivenTokenByUserID(t *testing.T) {
// 	userID, sessionID, _, err := addTestSession()
// 	require.NoError(t, err)

// 	err = DeleteLongLivenTokenByUserID(context.TODO(), sessionID)
// 	require.NoError(t, err)

// 	exists, err := db.Bun().NewSelect().Table("user_sessions").
// 		Where("user_id = ?", userID).Exists(context.TODO())
// 	require.False(t, exists)
// 	require.NoError(t, err)
// }

// func TestDeleteLongLivenTokenByToken(t *testing.T) {
// 	userID, _, token, err := addTestSession()
// 	require.NoError(t, err)

// 	err = DeleteLongLivenTokenByToken(context.TODO(), token)
// 	require.NoError(t, err)

// 	exists, err := db.Bun().NewSelect().Table("user_sessions").
// 		Where("user_id = ?", userID).Exists(context.TODO())
// 	require.False(t, exists)
// 	require.NoError(t, err)
// }

// func TestDeleteLongLivedTokenByTokenID(t *testing.T) {
// 	userID, sessionID, _, err := addTestSession()
// 	require.NoError(t, err)

// 	err = DeleteLongLivedTokenByTokenID(context.TODO(), sessionID)
// 	require.NoError(t, err)

// 	exists, err := db.Bun().NewSelect().Table("user_sessions").
// 		Where("user_id = ?", userID).Exists(context.TODO())
// 	require.False(t, exists)
// 	require.NoError(t, err)
// }
