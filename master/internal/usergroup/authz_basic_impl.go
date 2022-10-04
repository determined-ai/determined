package usergroup

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// UserGroupAuthZBasic is basic OSS controls.
type UserGroupAuthZBasic struct{}

// CanGetGroup always returns nil.
func (a *UserGroupAuthZBasic) CanGetGroup(ctx context.Context, curUser model.User, gid int) (bool, error) {
	return true, nil
}

// FilterGroupsList returns the list it was given and a nil error.
func (a *UserGroupAuthZBasic) FilterGroupsList(curUser model.User,
	query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	return query, nil
}

// CanUpdateGroups always returns nil.
func (a *UserGroupAuthZBasic) CanUpdateGroups(curUser model.User) (bool, error) {
	if curUser.Admin {
		return true, nil
	}
	return false, fmt.Errorf("access denied")
}

func init() {
	AuthZProvider.Register("basic", &UserGroupAuthZBasic{})
}
