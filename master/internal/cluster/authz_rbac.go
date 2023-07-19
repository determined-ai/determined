package cluster

import (
	"context"

	"github.com/determined-ai/determined/master/internal/rbac"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// MiscAuthZRBAC is the RBAC implementation of the MiscAuthZ interface.
type MiscAuthZRBAC struct{}

func (a *MiscAuthZRBAC) checkForPermission(
	ctx context.Context,
	curUser *model.User,
	permission rbacv1.PermissionType,
	options ...rbac.CheckForPermissionOptionsFunc,
) (permErr error, err error) {
	return rbac.CheckForPermission(
		ctx,
		"misc",
		curUser,
		nil,
		permission,
		options...,
	)
}

// CanUpdateAgents checks if the user can update agents.
func (a *MiscAuthZRBAC) CanUpdateAgents(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	return a.checkForPermission(
		ctx,
		curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_AGENTS)
}

// CanGetSensitiveAgentInfo checks if the user can view sensitive subset of agent info.
func (a *MiscAuthZRBAC) CanGetSensitiveAgentInfo(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	return a.checkForPermission(ctx,
		curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_SENSITIVE_AGENT_INFO,
		rbac.EnablePermissionCheckLogging(false),
	)
}

// CanGetMasterLogs checks if the user has permission to view master logs.
func (a *MiscAuthZRBAC) CanGetMasterLogs(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	return a.checkForPermission(
		ctx,
		curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_MASTER_LOGS,
	)
}

// CanGetMasterConfig checks if the user has permission to view master configs.
func (a *MiscAuthZRBAC) CanGetMasterConfig(
	ctx context.Context, curUser *model.User,
) (permErr error, error error) {
	return a.checkForPermission(
		ctx,
		curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_MASTER_CONFIG,
	)
}

// CanUpdateMasterConfig checks if the user has permission to view master configs.
func (a *MiscAuthZRBAC) CanUpdateMasterConfig(
	ctx context.Context, curUser *model.User,
) (permErr error, error error) {
	return a.checkForPermission(
		ctx,
		curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_MASTER_CONFIG,
	)
}

// CanGetUsageDetails checks if the user can get usage related details.
func (a *MiscAuthZRBAC) CanGetUsageDetails(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	return a.checkForPermission(
		ctx,
		curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_CLUSTER_USAGE,
	)
}

func init() {
	AuthZProvider.Register("rbac", &MiscAuthZRBAC{})
}
