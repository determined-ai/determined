package user

import (
	"context"
	"crypto/sha512"
	"fmt"
	"strings"
	"unicode"
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
	if err := CheckPasswordComplexity(password); err != nil {
		return err
	}

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

// PasswordComplexityErrors indicates that a password can't be set
// because it fails to meet current complexity requirements.
type PasswordComplexityErrors []complexityError

type complexityError int

const (
	errPasswordTooShort complexityError = iota
	errPasswordRequiresUppercase
	errPasswordRequiresLowercase
	errPasswordRequiresNumber
)

// Attempts to match the error strings in webui/react/src/components/UserSettings.tsx.
var complexityErrorString = map[complexityError]string{
	errPasswordTooShort:          "password must have at least 8 characters",
	errPasswordRequiresUppercase: "password must include an uppercase letter",
	errPasswordRequiresLowercase: "password must include a lowercase letter",
	errPasswordRequiresNumber:    "password must include a number",
}

// Error satisfies the error interface builtin.
func (e PasswordComplexityErrors) Error() string {
	switch len(e) {
	case 0:
		return "you found a bug! Please file a github issue citing error ba30317b-f59a-4c2b-9832-e7e36900bbda"
	case 1:
		return complexityErrorString[e[0]]
	default:
		errs := ""
		for _, err := range e {
			errs += fmt.Sprintf("%s\n", complexityErrorString[err])
		}
		return errs
	}
}

// CheckPasswordComplexity returns an error if the provided password does not satisfy
// current complexity requirements.
func CheckPasswordComplexity(password string) error {
	var err PasswordComplexityErrors
	if len(password) < 8 {
		err = append(err, errPasswordTooShort)
	}
	if !strings.ContainsFunc(password, unicode.IsUpper) {
		err = append(err, errPasswordRequiresUppercase)
	}
	if !strings.ContainsFunc(password, unicode.IsLower) {
		err = append(err, errPasswordRequiresLowercase)
	}
	if !strings.ContainsFunc(password, unicode.IsNumber) {
		err = append(err, errPasswordRequiresNumber)
	}
	if len(err) != 0 {
		return err
	}
	return nil
}
