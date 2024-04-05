//go:build integration
// +build integration

package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
)

func TestUpdateDefaultUserPasswords(t *testing.T) {
	u := db.RequireMockUser(t, db.SingleDB())
	err := SetUserPassword(context.Background(), u.Username, "abcDEF123!")
	require.NoError(t, err)

	nu, err := ByUsername(context.Background(), u.Username)
	require.NoError(t, err)
	ok := nu.ValidatePassword(ReplicateClientSideSaltAndHash("abcDEF123!"))
	require.True(t, ok, "couldn't login after default changes")
}

func TestUpdateUserPasswordComplexityCheck(t *testing.T) {
	u := db.RequireMockUser(t, db.SingleDB())

	// empty passwords are grandfathered in, for now
	nu, err := ByUsername(context.Background(), u.Username)
	require.NoError(t, err)
	ok := nu.ValidatePassword("")
	require.True(t, ok, "couldn't login after changes")

	// changing to a bad password is not allowed
	var complexityErr PasswordComplexityErrors
	err = SetUserPassword(context.Background(), u.Username, "")
	require.ErrorAs(t, err, &complexityErr)
	require.Len(t, complexityErr, 4)

	err = SetUserPassword(context.Background(), u.Username, "abc")
	require.ErrorAs(t, err, &complexityErr)
	require.Len(t, complexityErr, 3)
	require.Contains(t, complexityErr, errPasswordTooShort)
	require.Contains(t, complexityErr, errPasswordRequiresUppercase)
	require.Contains(t, complexityErr, errPasswordRequiresNumber)

	err = SetUserPassword(
		context.Background(),
		u.Username,
		"LEX LUTHOR TOOK 40 CAKES, THAT IS FOUR TENS, AND THAT'S TERRIBLE",
	)
	require.ErrorAs(t, err, &complexityErr)
	require.Len(t, complexityErr, 1)
	require.Contains(t, complexityErr, errPasswordRequiresLowercase)

	// changing to a good password is allowed
	err = SetUserPassword(context.Background(), u.Username, "Hunter2!")
	require.NoError(t, err)
}
