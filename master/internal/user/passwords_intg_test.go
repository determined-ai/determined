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
	err := SetUserPassword(context.Background(), u.Username, "abc")
	require.ErrorAs(t, err, &PasswordComplexityErrors{})

	// empty passwords are grandfathered in, for now
	nu, err := ByUsername(context.Background(), u.Username)
	require.NoError(t, err)
	ok := nu.ValidatePassword("")
	require.True(t, ok, "couldn't login after changes")
}
