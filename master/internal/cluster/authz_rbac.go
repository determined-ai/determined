package cluster

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// MiscAuthZRBAC is the RBAC implementation of the MiscAuthZ interface.
type MiscAuthZRBAC struct{}

func (a *MiscAuthZRBAC) checkForPermission(
	ctx context.Context, curUser *model.User, permission rbacv1.PermissionType,
) (permErr error, err error) {
	fields := audit.ExtractLogFields(ctx)
	fields["userID"] = curUser.ID
	fields["username"] = curUser.Username
	fields["permissionsRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{permission},
			SubjectType:     "misc",
			SubjectIDs:      []string{"master-logs"},
		},
	}

	defer func() {
		if err == nil {
			fields["permissionGranted"] = permErr == nil
			audit.Log(fields)
		}
	}()

	if err := db.DoesPermissionMatch(ctx, curUser.ID, nil,
		permission); err != nil {
		switch typedErr := err.(type) {
		case authz.PermissionDeniedError:
			return typedErr, nil
		default:
			return nil, err
		}
	}
	return nil, nil
}

// CanUpdateAgents checks if the user can update agents.
func (a *MiscAuthZRBAC) CanUpdateAgents(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	return a.checkForPermission(ctx, curUser, rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_AGENTS)
}

// CanGetMasterLogs checks if the user has permission to view master logs.
func (a *MiscAuthZRBAC) CanGetMasterLogs(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	return a.checkForPermission(ctx, curUser, rbacv1.PermissionType_PERMISSION_TYPE_VIEW_MASTER_LOGS)
}

// CanGetUsageDetails checks if the user can get usage related details.
func (a *MiscAuthZRBAC) CanGetUsageDetails(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	return a.checkForPermission(ctx, curUser, rbacv1.PermissionType_PERMISSION_TYPE_VIEW_CLUSTER_USAGE)
}

func init() {
	AuthZProvider.Register("rbac", &MiscAuthZRBAC{})
}
