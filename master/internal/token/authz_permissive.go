package token

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TokenAuthZPermissive is the permission implementation.
type TokenAuthZPermissive struct{}

// CanCreateAccessToken calls RBAC authz but enforces basic authz.
func (p *TokenAuthZPermissive) CanCreateAccessToken(
	ctx context.Context, curUser, targetUser model.User,
) error {
	_ = (&TokenAuthZRBAC{}).CanCreateAccessToken(ctx, curUser, targetUser)
	return (&TokenAuthZBasic{}).CanCreateAccessToken(ctx, curUser, targetUser)
}

// CanGetAccessTokens calls RBAC authz but enforces basic authz.
func (p *TokenAuthZPermissive) CanGetAccessTokens(
	ctx context.Context, curUser model.User, query *bun.SelectQuery, targetUserID *model.UserID,
) (*bun.SelectQuery, error) {
	_, _ = (&TokenAuthZRBAC{}).CanGetAccessTokens(ctx, curUser, query, targetUserID)
	return (&TokenAuthZBasic{}).CanGetAccessTokens(ctx, curUser, query, targetUserID)
}

// CanUpdateAccessToken calls RBAC authz but enforces basic authz.
func (p *TokenAuthZPermissive) CanUpdateAccessToken(
	ctx context.Context,
	curUser model.User,
	targetTokenUserID model.UserID,
) error {
	_ = (&TokenAuthZRBAC{}).CanUpdateAccessToken(ctx, curUser, targetTokenUserID)
	return (&TokenAuthZBasic{}).CanUpdateAccessToken(ctx, curUser, targetTokenUserID)
}

func init() {
	AuthZProvider.Register("permissive", &TokenAuthZPermissive{})
}
