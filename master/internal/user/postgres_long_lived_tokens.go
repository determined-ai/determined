package user

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/o1egl/paseto"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// DefaultTokenLifespan is how long a newly created long lived token is valid.
const DefaultTokenLifespan = 30 * 24 * time.Hour

// CurrentTimeNowInUTC stores the current time in UTC for time insertions.
var CurrentTimeNowInUTC time.Time

// LongLivedTokenOption is the return type for WithTokenExpiresAt helper function.
// It takes a pointer to model.LongLivedToken and modifies it.
// It’s used to apply optional settings to the LongLivedToken object.
type LongLivedTokenOption func(f *model.LongLivedToken)

// WithTokenExpiresAt function will add specified expiresAt (if any) to the long lived token table.
func WithTokenExpiresAt(expiresAt *time.Duration) LongLivedTokenOption {
	return func(s *model.LongLivedToken) {
		s.ExpiresAt = CurrentTimeNowInUTC.Add(*expiresAt)
	}
}

// DeleteAndCreateLongLivedToken creates a row in the long lived token table.
func DeleteAndCreateLongLivedToken(
	ctx context.Context, userID model.UserID, opts ...LongLivedTokenOption,
) (string, error) {
	CurrentTimeNowInUTC = time.Now().UTC()
	// Populate the default values in the model.
	longLivedToken := &model.LongLivedToken{
		UserID:    userID,
		CreatedAt: CurrentTimeNowInUTC,
		ExpiresAt: CurrentTimeNowInUTC.Add(DefaultTokenLifespan),
	}

	// Update the optional ExpiresAt field (if passed)
	for _, opt := range opts {
		opt(longLivedToken)
	}

	var token string

	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Tokens should have a 1:1 relationship with users, if a user creates a new token,
		// revoke the previous token if it exists.
		_, err := tx.NewDelete().
			Table("long_lived_tokens").
			Where("user_id = ?", userID).
			Exec(ctx)
		if err != nil {
			return err
		}

		// A new row is inserted into the long_lived_token table, and the ID of the
		// inserted row is returned and stored in longLivedToken.ID.
		_, err = tx.NewInsert().
			Model(longLivedToken).
			Column("user_id", "expires_at", "created_at").
			Returning("id").
			Exec(ctx, &longLivedToken.ID)
		if err != nil {
			return err
		}

		// A Paseto token is generated using the longLivedToken object and the private key.
		v2 := paseto.NewV2()
		privateKey := db.GetTokenKeys().PrivateKey
		token, err = v2.Sign(privateKey, longLivedToken, nil)
		if err != nil {
			return fmt.Errorf("failed to generate user authentication token: %s", err)
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return token, nil
}

// DeleteLongLivenTokenByUserID deletes long lived token if found.
// If not found, the err will be nil, and the number of affected rows will be zero.
func DeleteLongLivenTokenByUserID(ctx context.Context, userID model.UserID) error {
	res, err := db.Bun().NewDelete().
		Table("long_lived_tokens").
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		return err // Return error if the delete operation itself failed
	}

	// Check how many rows were affected
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err // Return error if checking the rows affected failed
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows found with user_id: %v", userID) // Custom error when no rows were found
	}

	return nil // Return nil if deletion was successful
}

// DeleteLongLivenTokenByToken deletes long lived token if found.
func DeleteLongLivenTokenByToken(ctx context.Context, token string) error {
	v2 := paseto.NewV2()
	var longLivedToken model.LongLivedToken
	// Verification will fail when using external token (Jwt instead of Paseto).
	// Currently passing Paseto token.
	if err := v2.Verify(token, db.GetTokenKeys().PublicKey, &longLivedToken, nil); err != nil {
		return nil //nolint: nilerr
	}
	return DeleteLongLivedTokenByTokenID(ctx, longLivedToken.ID)
}

// DeleteLongLivedTokenByTokenID deletes the long lived token with the given token ID.
// If not found, the err will be nil, and the number of affected rows will be zero.
func DeleteLongLivedTokenByTokenID(ctx context.Context, longLivedTokenID model.TokenID) error {
	res, err := db.Bun().NewDelete().
		Table("long_lived_tokens").
		Where("id = ?", longLivedTokenID).
		Exec(ctx)
	if err != nil {
		return err // Return error if the delete operation itself failed
	}

	// Check how many rows were affected
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err // Return error if checking the rows affected failed
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows found with token_id: %v", longLivedTokenID) // Custom error when no rows were found
	}

	return nil // Return nil if deletion was successful
}

// GetLongLivedTokenInfo returns the token info from the table with the given user_id.
func GetLongLivedTokenInfo(ctx context.Context, userID model.UserID) (
	*model.LongLivedToken, error,
) {
	var tokenInfo model.LongLivedToken // To store the token info for the given user_id

	// Execute the query to fetch the token info for the given user_id
	switch err := db.Bun().NewSelect().Table("long_lived_tokens").
		Where("user_id = ?", userID).
		Scan(ctx, &tokenInfo); {
	case errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("no rows found with user_id: %v", userID)
	case err != nil:
		return nil, err
	default:
		return &tokenInfo, nil
	}
}
