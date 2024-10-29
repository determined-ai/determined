package token

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TokenAuthZ describes authz methods for `accesstoken` package.
type TokenAuthZ interface {
	// POST /api/v1/users/:user_id/token
	CanCreateAccessToken(ctx context.Context, curUser, targetUser model.User) error
	// GET /api/v1/user/tokens
	CanGetAccessTokens(ctx context.Context, curUser model.User, query *bun.SelectQuery,
		targetUserID *model.UserID) (*bun.SelectQuery, error)
	// PATCH /api/v1/users/token/:token_id
	CanUpdateAccessToken(ctx context.Context, curUser model.User, targetTokenUserID model.UserID) error
}

// AuthZProvider is the authz registry for `token` package.
var AuthZProvider authz.AuthZProviderType[TokenAuthZ]
