package rbac

import (
	"fmt"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/uptrace/bun"
)

// RBACAuthZBasic is basic OSS controls.
type RBACAuthZBasic struct{}

// CanGetRoles always returns nil.
func (a *RBACAuthZBasic) CanGetRoles(curUser model.User) error {
	return nil
}

// FilterRolesQuery always returns nil.
func (a *RBACAuthZBasic) FilterRolesQuery(curUser model.User, query *bun.SelectQuery) (*bun.SelectQuery, error) {
	return query, nil
}

// CanGetUserRoles always returns nil.
func (a *RBACAuthZBasic) CanGetUserRoles(curUser model.User) error {
	return nil
}

// CanGetGroupRoles always returns nil.
func (a *RBACAuthZBasic) CanGetGroupRoles(curUser model.User) error {
	return nil
}

// CanSearchScope always returns nil.
func (a *RBACAuthZBasic) CanSearchScope(curUser model.User) error {
	return nil
}

// CanAssignRoles always returns nil.
func (a *RBACAuthZBasic) CanAssignRoles(curUser model.User) error {
	if curUser.Admin {
		return nil
	}
	return fmt.Errorf("access denied")
}

// CanRemoveRoles always returns nil.
func (a *RBACAuthZBasic) CanRemoveRoles(curUser model.User) error {
	return a.CanAssignRoles(curUser)
}