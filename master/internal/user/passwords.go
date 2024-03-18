package user

import (
	"context"
	"crypto/sha512"
	"fmt"
)

const (
	determinedUsername = "determined"
	adminUsername      = "admin"
)

// BuiltInUsers are created in the DB by the initial migration. They exist on every installation unless the
// admin has removed them.
var BuiltInUsers = []string{determinedUsername, adminUsername}

const clientSidePasswordSalt = "GubPEmmotfiK9TMD6Zdw" // #nosec G101

// ReplicateClientSideSaltAndHash replicates the password salt and hash done on the client side.
// We need this because we hash passwords on the client side, but when SCIM posts a user with
// a password to password sync, it doesn't - so when we try to log in later, we get a weird,
// unrecognizable sha512 hash from the frontend.
func ReplicateClientSideSaltAndHash(password string) string {
	if password == "" {
		return password
	}
	sum := sha512.Sum512([]byte(clientSidePasswordSalt + password))
	return fmt.Sprintf("%x", sum)
}

// SetUserPassword sets the password of the user with the given username to the plaintext string provided.
func SetUserPassword(ctx context.Context, username, password string) error {
	u, err := ByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("retrieving user %s: %w", username, err)
	}

	err = u.UpdatePasswordHash(ReplicateClientSideSaltAndHash(password))
	if err != nil {
		return fmt.Errorf("updating password hash for user %s: %w", username, err)
	}

	err = Update(ctx, u, []string{"password_hash"}, nil)
	if err != nil {
		return fmt.Errorf("updating password hash for user %s: %w", username, err)
	}
	return nil
}
