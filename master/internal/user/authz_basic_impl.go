package user

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/model"
)

// UserAuthZBasic is basic.
type UserAuthZBasic struct{}

// CanSetUserPassword for basic authz.
func (a *UserAuthZBasic) CanSetUserPassword(currentUser model.User, targetUser model.User) error {
	if !currentUser.Admin && currentUser.ID != targetUser.ID {
		return status.Error(
			codes.PermissionDenied, "non-admin users can only change their own password")
	}
	return nil
}

// CanSetUserDisplayName for basic authz.
func (a *UserAuthZBasic) CanSetUserDisplayName(
	currentUser model.User, targetUser model.User) error {
	if !currentUser.Admin && currentUser.ID != targetUser.ID {
		return status.Error(
			codes.PermissionDenied, "non-admin users can only change their own display name")
	}
	return nil
}

// CanSetUserActive for basic authz.
func (a *UserAuthZBasic) CanSetUserActive(currentUser model.User, targetUser model.User) error {
	if !currentUser.Admin {
		return status.Error(
			codes.PermissionDenied, "only admin can activate/deactivate user")
	}
	return nil
}

// CanSetUserAdmin for basic authz.
func (a *UserAuthZBasic) CanSetUserAdmin(currentUser model.User, targetUser model.User) error {
	if !currentUser.Admin {
		return status.Error(
			codes.PermissionDenied, "only admin can change user from/to admin")
	}
	return nil
}

// CanSetUserAgentGroup for basic authz.
func (a *UserAuthZBasic) CanSetUserAgentGroup(currentUser model.User, targetUser model.User) error {
	if !currentUser.Admin {
		return status.Error(codes.PermissionDenied, "only admin can set user agent group")
	}
	return nil
}

func init() {
	AuthZProvider.Register("basic", &UserAuthZBasic{})
}
