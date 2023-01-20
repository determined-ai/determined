package user

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
)

// UserAuthZPermissive is the permission implementation.
type UserAuthZPermissive struct{}

// CanGetUser calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanGetUser(
	ctx context.Context, curUser, targetUser model.User,
) (bool, error) {
	_, _ = (&UserAuthZRBAC{}).CanGetUser(ctx, curUser, targetUser)
	return (&UserAuthZBasic{}).CanGetUser(ctx, curUser, targetUser)
}

// FilterUserList calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) FilterUserList(
	ctx context.Context, curUser model.User, users []model.FullUser,
) ([]model.FullUser, error) {
	_, _ = (&UserAuthZRBAC{}).FilterUserList(ctx, curUser, users)
	return (&UserAuthZBasic{}).FilterUserList(ctx, curUser, users)
}

// CanCreateUser calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanCreateUser(
	ctx context.Context, curUser, userToAdd model.User,
	agentUserGroup *model.AgentUserGroup,
) error {
	_ = (&UserAuthZRBAC{}).CanCreateUser(ctx, curUser, userToAdd, agentUserGroup)
	return (&UserAuthZBasic{}).CanCreateUser(ctx, curUser, userToAdd, agentUserGroup)
}

// CanSetUsersPassword calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanSetUsersPassword(
	ctx context.Context, curUser, targetUser model.User,
) error {
	_ = (&UserAuthZRBAC{}).CanSetUsersPassword(ctx, curUser, targetUser)
	return (&UserAuthZBasic{}).CanSetUsersPassword(ctx, curUser, targetUser)
}

// CanSetUsersActive calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanSetUsersActive(
	ctx context.Context, curUser, targetUser model.User, toActiveVal bool,
) error {
	_ = (&UserAuthZRBAC{}).CanSetUsersActive(ctx, curUser, targetUser, toActiveVal)
	return (&UserAuthZBasic{}).CanSetUsersActive(ctx, curUser, targetUser, toActiveVal)
}

// CanSetUsersAdmin calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanSetUsersAdmin(
	ctx context.Context, curUser, targetUser model.User, toAdminVal bool,
) error {
	_ = (&UserAuthZRBAC{}).CanSetUsersAdmin(ctx, curUser, targetUser, toAdminVal)
	return (&UserAuthZBasic{}).CanSetUsersAdmin(ctx, curUser, targetUser, toAdminVal)
}

// CanSetUsersAgentUserGroup calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanSetUsersAgentUserGroup(
	ctx context.Context, curUser, targetUser model.User,
	agentUserGroup model.AgentUserGroup,
) error {
	_ = (&UserAuthZRBAC{}).CanSetUsersAgentUserGroup(ctx, curUser, targetUser, agentUserGroup)
	return (&UserAuthZBasic{}).CanSetUsersAgentUserGroup(ctx, curUser, targetUser, agentUserGroup)
}

// CanSetUsersUsername calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanSetUsersUsername(
	ctx context.Context, curUser, targetUser model.User,
) error {
	_ = (&UserAuthZRBAC{}).CanSetUsersUsername(ctx, curUser, targetUser)
	return (&UserAuthZBasic{}).CanSetUsersUsername(ctx, curUser, targetUser)
}

// CanSetUsersDisplayName calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanSetUsersDisplayName(
	ctx context.Context, curUser, targetUser model.User,
) error {
	_ = (&UserAuthZRBAC{}).CanSetUsersDisplayName(ctx, curUser, targetUser)
	return (&UserAuthZBasic{}).CanSetUsersDisplayName(ctx, curUser, targetUser)
}

// CanGetUsersImage calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanGetUsersImage(
	ctx context.Context, curUser, targetUser model.User,
) error {
	_ = (&UserAuthZRBAC{}).CanGetUsersImage(ctx, curUser, targetUser)
	return (&UserAuthZBasic{}).CanGetUsersImage(ctx, curUser, targetUser)
}

// CanGetUsersOwnSettings calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanGetUsersOwnSettings(
	ctx context.Context, curUser model.User,
) error {
	_ = (&UserAuthZRBAC{}).CanGetUsersOwnSettings(ctx, curUser)
	return (&UserAuthZBasic{}).CanGetUsersOwnSettings(ctx, curUser)
}

// CanCreateUsersOwnSetting calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanCreateUsersOwnSetting(
	ctx context.Context, curUser model.User, setting model.UserWebSetting,
) error {
	_ = (&UserAuthZRBAC{}).CanCreateUsersOwnSetting(ctx, curUser, setting)
	return (&UserAuthZBasic{}).CanCreateUsersOwnSetting(ctx, curUser, setting)
}

// CanResetUsersOwnSettings calls RBAC authz but enforces basic authz.
func (p *UserAuthZPermissive) CanResetUsersOwnSettings(
	ctx context.Context, curUser model.User,
) error {
	_ = (&UserAuthZRBAC{}).CanResetUsersOwnSettings(ctx, curUser)
	return (&UserAuthZBasic{}).CanResetUsersOwnSettings(ctx, curUser)
}

func init() {
	AuthZProvider.Register("permissive", &UserAuthZPermissive{})
}
