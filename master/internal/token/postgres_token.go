package token

import (
	"context"
	"fmt"
	"time"

	"github.com/o1egl/paseto"
	"github.com/uptrace/bun"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	// DefaultTokenLifespan is how long a newly created access token is valid.
	DefaultTokenLifespan = 30 * 24 * time.Hour
)

// CurrentTimeNowInUTC stores the current time in UTC for time insertions.
var CurrentTimeNowInUTC time.Time

// AccessTokenOption is the return type for WithTokenExpiry helper function.
// It takes a pointer to model.UserSession and modifies it.
// It’s used to apply optional settings to the AccessToken object.
type AccessTokenOption func(f *model.UserSession)

// WithTokenExpiry function will add specified expiresAt (if any) to the access token table.
func WithTokenExpiry(expiry *time.Duration) AccessTokenOption {
	return func(s *model.UserSession) {
		s.Expiry = CurrentTimeNowInUTC.Add(*expiry)
	}
}

// WithTokenDescription function will add specified description (if any) to the access token table.
func WithTokenDescription(description string) AccessTokenOption {
	return func(s *model.UserSession) {
		if description == "" {
			return
		}
		s.Description = null.StringFrom(description)
	}
}

// CreateAccessToken creates a new access token and store in
// user_sessions db.
func CreateAccessToken(
	ctx context.Context, userID model.UserID, opts ...AccessTokenOption,
) (string, model.TokenID, error) {
	CurrentTimeNowInUTC = time.Now().UTC()
	// Populate the default values in the model.
	accessToken := &model.UserSession{
		UserID:      userID,
		CreatedAt:   CurrentTimeNowInUTC,
		Expiry:      CurrentTimeNowInUTC.Add(DefaultTokenLifespan),
		TokenType:   model.TokenTypeAccessToken,
		Description: null.StringFromPtr(nil),
		Revoked:     false,
	}

	// Update the optional ExpiresAt field (if passed)
	for _, opt := range opts {
		opt(accessToken)
	}

	var token string

	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// AccessTokens should have a many:1 relationship with users.
		// A new row is inserted into the user_sessions table, and the ID of the
		// inserted row is returned and stored in user_sessions.ID.
		_, err := tx.NewInsert().
			Model(accessToken).
			Column("user_id", "expiry", "created_at", "token_type", "revoked", "description").
			Returning("id").
			Exec(ctx, &accessToken.ID)
		if err != nil {
			return err
		}

		// A Paseto token is generated using the accessToken object and the private key.
		v2 := paseto.NewV2()
		privateKey := db.GetTokenKeys().PrivateKey
		token, err = v2.Sign(privateKey, accessToken, nil)
		if err != nil {
			return fmt.Errorf("failed to generate user authentication token: %s", err)
		}

		return nil
	})
	if err != nil {
		return "", 0, err
	}

	return token, model.TokenID(accessToken.ID), nil
}

// AccessTokenUpdateOptions is the set of mutable fields for an Access Token record.
type AccessTokenUpdateOptions struct {
	Description *string
	SetRevoked  bool
}

// UpdateAccessToken updates the description and revocation status of the access token.
func UpdateAccessToken(
	ctx context.Context, tokenID model.TokenID, options AccessTokenUpdateOptions,
) (*model.UserSession, error) {
	var tokenInfo model.UserSession
	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewSelect().Table("user_sessions").
			Where("id = ?", tokenID).Where("token_type = ?", model.TokenTypeAccessToken).
			Scan(ctx, &tokenInfo)
		if err != nil {
			return err
		}

		if tokenInfo.Revoked || tokenInfo.Expiry.Before(time.Now().UTC()) {
			return fmt.Errorf("unable to update inactive token with ID %v", tokenID)
		}

		if options.Description != nil {
			tokenInfo.Description = null.StringFrom(*options.Description)
		}

		if options.SetRevoked {
			tokenInfo.Revoked = true
		}

		_, err = tx.NewUpdate().
			Model(&tokenInfo).
			Column("description", "revoked").
			Where("id = ?", tokenID).
			Exec(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &tokenInfo, nil
}

func GetUserIDFromTokenID(ctx context.Context, tokenID int32) (model.UserID, error) {
	var userID model.UserID
	err := db.Bun().NewSelect().
		Table("user_sessions").
		Column("user_id").
		Where("id = ?", tokenID).
		Scan(ctx, &userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}
