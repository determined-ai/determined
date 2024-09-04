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

const (
	// TokenExpirationDuration is how long a newly created long lived token is valid.
	TokenExpirationDuration = 30 * 24 * time.Hour
)

// LongLivedTokenOption is the return type for WithTokenExpiresAt helper function.
// It takes a pointer to model.LongLivedToken and modifies it.
// It’s used to apply optional settings to the LongLivedToken object.
type LongLivedTokenOption func(f *model.LongLivedToken)

// WithTokenExpiresAt function will add specified expiresAt (if any) to the long lived token table.
func WithTokenExpiresAt(expiresAt *time.Time) LongLivedTokenOption {
	return func(s *model.LongLivedToken) {
		s.ExpiresAt = *expiresAt
	}
}

// DeleteAndCreateLongLivedToken creates a row in the long lived token table.
func DeleteAndCreateLongLivedToken(
	ctx context.Context, userID model.UserID, opts ...LongLivedTokenOption,
) (string, error) {
	// Populate the default values in the model.
	longLivedToken := &model.LongLivedToken{
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(TokenExpirationDuration),
	}

	// Update the optional ExpiresAt field (if passed)
	for _, opt := range opts {
		opt(longLivedToken)
	}

	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Tokens should have a 1:1 relationship with users, if a user creates a new token,
		// revoke the previous token if it exists.
		err := DeleteLongLivenTokenByUserID(ctx, userID)
		if err != nil {
			return err
		}

		// A new row is inserted into the long_lived_token table, and the ID of the
		// inserted row is returned and stored in longLivedToken.ID.
		_, err = db.Bun().NewInsert().
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
		token, err := v2.Sign(privateKey, longLivedToken, nil)
		if err != nil {
			return fmt.Errorf("failed to generate user authentication token: %s", err)
		}
		longLivedToken.TokenValue = token

		return nil
	})
	if err != nil {
		return "", err
	}

	return longLivedToken.TokenValue, nil
}

// DeleteLongLivenTokenByUserID deletes long lived token if found.
// If not found, the err will be nil, and the number of affected rows will be zero.
func DeleteLongLivenTokenByUserID(ctx context.Context, userID model.UserID) error {
	_, err := db.Bun().NewDelete().
		Table("long_lived_tokens").
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
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
	_, err := db.Bun().NewDelete().
		Table("long_lived_tokens").
		Where("id = ?", longLivedTokenID).
		Exec(ctx)
	return err
}

func GetLongLivedTokenInfo(ctx context.Context, userID model.UserID) (
	*model.LongLivedToken, error,
) {
	var tokenInfo model.LongLivedToken // To store the token info for the given user_id

	// Execute the query to fetch the token info for the given user_id
	switch err := db.Bun().NewSelect().Table("long_lived_tokens").
		Where("user_id = ?", userID).
		Scan(ctx, &tokenInfo); {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &tokenInfo, nil
	}
}
