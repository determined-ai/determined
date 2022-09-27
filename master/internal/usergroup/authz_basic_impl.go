package usergroup

import (
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// UserGroupAuthZBasic is basic OSS controls.
type UserGroupAuthZBasic struct{}

// CanGetGroup always returns nil.
func (a *UserGroupAuthZBasic) CanGetGroup(curUser model.User, gid int) error {
	return nil
}

// FilterGroupsList returns the list it was given and a nil error.
func (a *UserGroupAuthZBasic) FilterGroupsList(curUser model.User,
	query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	return query, nil
}

// CanUpdateGroups always returns nil.
func (a *UserGroupAuthZBasic) CanUpdateGroups(curUser model.User) error {
	if curUser.Admin {
		return nil
	}
	return fmt.Errorf("non admin users may not update groups")
}

func init() {
	AuthZProvider.Register("basic", &UserGroupAuthZBasic{})
}
