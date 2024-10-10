package token

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TokenAuthZBasic is basic OSS controls.
type TokenAuthZBasic struct{}

// CanCreateAccessToken returns an error if the user is not an admin.
func (a *TokenAuthZBasic) CanCreateAccessToken(
	ctx context.Context, curUser, targetUser model.User,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can create token for own/other users")
	}
	return nil
}

// CanGetAccessTokens returns an error if the user does not have permission to view own or
// another user's token based on own role.
func (a *TokenAuthZBasic) CanGetAccessTokens(
	ctx context.Context, curUser model.User, query *bun.SelectQuery, targetUserID model.UserID,
) (selectQuery *bun.SelectQuery, err error) {
	err = canGetOthersAccessToken(ctx, curUser)
	if err != nil {
		if targetUserID > 0 && targetUserID != curUser.ID {
			return nil, err
		}
		err = canGetOwnAccessToken(ctx, curUser)
		if err != nil {
			return nil, err
		}
		query = query.Where("us.user_id = ?", curUser.ID)
	}
	return query, nil
}

// canGetOthersAccessTokens returns an error if the user is not an admin.
func canGetOthersAccessToken(ctx context.Context, curUser model.User) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can view other users")
	}
	return nil
}

// canGetOwnAccessTokens returns an error if the user is not an admin.
func canGetOwnAccessToken(
	ctx context.Context, curUser model.User,
) error {
	return nil
}

// CanUpdateAccessToken returns an error if the user is not an admin when attempting to update
// another user's token; otherwise, it returns nil.
func (a *TokenAuthZBasic) CanUpdateAccessToken(
	ctx context.Context,
	curUser model.User,
	targetTokenUserID model.UserID,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can update access tokens")
	}
	return nil
}

func init() {
	AuthZProvider.Register("basic", &TokenAuthZBasic{})
}
