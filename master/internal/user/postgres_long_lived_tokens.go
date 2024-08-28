package user

import (
	"context"
	"time"

	"github.com/o1egl/paseto"
	"github.com/uptrace/bun"
	"gopkg.in/guregu/null.v3"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	// TokenExpirationDuration is how long a newly created long lived token is valid.
	TokenExpirationDuration = 30 * 24 * time.Hour
)

// LongLivedTokenOption is the return type for WithTokenExpiryTime helper function.
// It takes a pointer to model.LongLivedToken and modifies it.
// It’s used to apply optional settings to the LongLivedToken object.
type LongLivedTokenOption func(f *model.LongLivedToken)

// WithTokenExpiryTime function will add specified expiryTime (if any) to the long lived token table.
func WithTokenExpiryTime(expiryTime *time.Time) LongLivedTokenOption {
	return func(s *model.LongLivedToken) {
		s.ExpiryTime = *expiryTime
	}
}

// CreateLongLivedToken creates a row in the long lived token table.
func CreateLongLivedToken(ctx context.Context, user *model.User, opts ...LongLivedTokenOption) (string, error) {
	// Populate the default values in the model.
	longLivedToken := &model.LongLivedToken{
		UserID:     user.ID,
		CreatedAt:  time.Now(),
		ExpiryTime: time.Now().Add(TokenExpirationDuration),
	}

	// Update the optional ExpiryTime field (if passed)
	for _, opt := range opts {
		opt(longLivedToken)
	}

	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// A new row is inserted into the long_lived_token table, and the ID of the
		// inserted row is returned and stored in longLivedToken.ID.
		_, err := db.Bun().NewInsert().
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
			return errors.Wrap(err, "failed to generate user authentication token")
		}

		// The token is hashed using model.HashPassword
		longLivedToken.TokenValue = token
		hashedToken, err := model.HashPassword(token)
		if err != nil {
			return errors.Wrap(err, "error updating user long lived token")
		}
		longLivedToken.TokenValueHash = null.StringFrom(hashedToken)

		// The TokenValueHash is updated in the database.
		_, err = db.Bun().NewUpdate().
			Model(longLivedToken).
			Column("token_value_hash").
			Where("id = (?)", longLivedToken.ID).
			Exec(ctx)
		if err != nil {
			return err
		}

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
