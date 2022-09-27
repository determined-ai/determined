package usergroup

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

// UserGroupAuthZBasic is basic OSS controls.
type UserGroupAuthZBasic struct{}

// CanGetGroup always returns nil.
func (a *UserGroupAuthZBasic) CanGetGroup(curUser model.User) error {
	return nil
}

// FilterGroupsList returns the list it was given and a nil error.
func (a *UserGroupAuthZBasic) FilterGroupsList(curUser model.User,
	groups []*groupv1.GroupSearchResult,
) ([]*groupv1.GroupSearchResult, error) {
	return groups, nil
}

// CanCreateGroups always returns nil.
func (a *UserGroupAuthZBasic) CanCreateGroups(curUser model.User) error {
	return nil
}

// CanUpdateGroup always returns nil.
func (a *UserGroupAuthZBasic) CanUpdateGroup(curUser model.User) error {
	return nil
}

// CanDeleteGroup always returns nil.
func (a *UserGroupAuthZBasic) CanDeleteGroup(curUser model.User) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &UserGroupAuthZBasic{})
}
