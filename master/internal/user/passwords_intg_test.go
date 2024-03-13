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
	err := SetUserPassword(context.Background(), u.Username, "abc")
	require.NoError(t, err)

	nu, err := ByUsername(context.Background(), u.Username)
	require.NoError(t, err)
	ok := nu.ValidatePassword(ReplicateClientSideSaltAndHash("abc"))
	require.True(t, ok, "couldn't login after default changes")
}
