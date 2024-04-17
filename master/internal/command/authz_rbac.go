package command

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

// NSCAuthZRBAC is the RBAC implementation of the NSCAuthZ interface.
type NSCAuthZRBAC struct{}

func (a *NSCAuthZRBAC) addLogInfo(
	fields *log.Fields, curUser model.User, permission rbacv1.PermissionType,
	targetWorkscapeID model.AccessScopeID,
) {
	if fields == nil {
		return
	}
	// NSC ID is set by the caller at fields[audit.EntityIDKey].
	(*fields)["userID"] = curUser.ID
	(*fields)["permissionsRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{permission},
			SubjectType:     "NSC Workspace",
			SubjectIDs:      []string{fmt.Sprint(targetWorkscapeID)},
		},
	}
}

func (a *NSCAuthZRBAC) addLogInfoWorkspaces(
	fields *log.Fields, curUser model.User, permission rbacv1.PermissionType,
	targetWorkspaceIDs []model.AccessScopeID,
) {
	if fields == nil {
		return
	}

	var subjectIDs []string
	for _, scope := range targetWorkspaceIDs {
		subjectIDs = append(subjectIDs, fmt.Sprint(scope))
	}

	// NSC ID is set by the caller at fields[audit.EntityIDKey].
	(*fields)["userID"] = curUser.ID
	(*fields)["permissionsRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{permission},
			SubjectType:     "NSC Workspace",
			SubjectIDs:      subjectIDs,
		},
	}
}

func (a *NSCAuthZRBAC) checkForPermissions(
	ctx context.Context, curUser model.User, workspaceIDs []model.AccessScopeID,
	permission rbacv1.PermissionType,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	a.addLogInfoWorkspaces(&fields, curUser, permission, workspaceIDs)
	defer func() {
		if err == nil || authz.IsPermissionDenied(err) {
			fields["permissionGranted"] = !authz.IsPermissionDenied(err)
			audit.Log(fields)
		}
	}()
	wids := []int32{}
	for _, id := range workspaceIDs {
		wids = append(wids, int32(id))
	}
	return db.DoesPermissionMatchAll(ctx, curUser.ID, permission, wids...)
}

func (a *NSCAuthZRBAC) checkForPermission(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	permission rbacv1.PermissionType,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	a.addLogInfo(&fields, curUser, permission, workspaceID)
	defer func() {
		if err == nil || authz.IsPermissionDenied(err) {
			fields["permissionGranted"] = !authz.IsPermissionDenied(err)
			audit.Log(fields)
		}
	}()

	wID := int32(workspaceID)
	err = db.DoesPermissionMatch(ctx, curUser.ID, &wID,
		permission)
	return err
}

// CanGetNSC checks if the user is authorized to view NSCs in the specified workspace.
func (a *NSCAuthZRBAC) CanGetNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	return a.checkForPermission(ctx, curUser, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_NSC)
}

// CanGetActiveTasksCount always returns a nil error.
func (a *NSCAuthZRBAC) CanGetActiveTasksCount(ctx context.Context, curUser model.User) (err error) {
	return nil
}

// CanTerminateNSC checks if the user is authorized to terminate NSCs in the workspace.
func (a *NSCAuthZRBAC) CanTerminateNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) (err error) {
	return a.checkForPermission(ctx, curUser, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_NSC)
}

// CanCreateNSC checks if the user is authorized to create NSCs in the workspace.
func (a *NSCAuthZRBAC) CanCreateNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	// TODO(DET-8774): the caller should check for workspace status (archived).
	return a.checkForPermission(ctx, curUser, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_NSC)
}

// CanSetNSCsPriority checks if the user is authorized to set NSCs priority in the workspace.
func (a *NSCAuthZRBAC) CanSetNSCsPriority(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID, priority int,
) error {
	// CHECK(DET-8794): we only just need workspaceID here.
	return a.checkForPermission(ctx, curUser, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_NSC)
}

// AccessibleScopes returns the set of scopes that the user should be limited to.
func (a *NSCAuthZRBAC) AccessibleScopes(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
) (model.AccessScopeSet, error) {
	return rbac.PermittedScopes(ctx, curUser, requestedScope,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_NSC)
}

// FilterTensorboards returns the tensorboards that the user has access to.
func (a *NSCAuthZRBAC) FilterTensorboards(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
	tensorboards []*tensorboardv1.Tensorboard,
) ([]*tensorboardv1.Tensorboard, error) {
	var filteredTensorboards []*tensorboardv1.Tensorboard
	filteredScopes, err := rbac.PermittedScopes(ctx, curUser, requestedScope,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_ARTIFACTS)
	if err != nil {
		return nil, err
	}

	for _, tb := range tensorboards {
		workspaceIDs, err := tensorboardWorkspaces(ctx, tb.ExperimentIds, tb.TrialIds)
		if err != nil {
			continue
		}
		accessGranted := true
		if _, ok := filteredScopes[model.AccessScopeID(tb.WorkspaceId)]; !ok {
			continue
		}
		for _, wID := range workspaceIDs {
			if _, ok := filteredScopes[wID]; !ok {
				accessGranted = false
				break
			}
		}
		if accessGranted {
			filteredTensorboards = append(filteredTensorboards, tb)
		}
	}

	return filteredTensorboards, nil
}

// CanGetTensorboard returns whether or not a user owns the tensorboard or can access it.
func (a *NSCAuthZRBAC) CanGetTensorboard(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	experimentIDs []int32, trialIDs []int32,
) error {
	err := a.checkForPermission(ctx, curUser, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_WORKSPACE)
	if err != nil {
		return err
	}

	expToWorkspaceIDs, err := db.ExperimentIDsToWorkspaceIDs(ctx, experimentIDs)
	if err != nil {
		return err
	}

	trialsToWorkspaceIDs, err := db.TrialIDsToWorkspaceIDs(ctx, trialIDs)
	if err != nil {
		return err
	}

	var workspaceIDs []model.AccessScopeID
	workspaceIDs = append(workspaceIDs, expToWorkspaceIDs...)
	workspaceIDs = append(workspaceIDs, trialsToWorkspaceIDs...)

	if len(workspaceIDs) == 0 {
		return nil
	}

	err = a.checkForPermissions(ctx, curUser,
		workspaceIDs, rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_ARTIFACTS)

	return err
}

// CanTerminateTensorboard always returns nil.
func (a *NSCAuthZRBAC) CanTerminateTensorboard(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	return a.CanTerminateNSC(ctx, curUser, workspaceID)
}

// CanCreateGenericTask returns whether or not a user can create a generic task.
func (a *NSCAuthZRBAC) CanCreateGenericTask(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	return a.checkForPermission(ctx, curUser, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_NSC)
}

func tensorboardWorkspaces(
	ctx context.Context, experimentIDs []int32, trialIDs []int32,
) ([]model.AccessScopeID, error) {
	expIDsToWorkspaceIDs, err := db.ExperimentIDsToWorkspaceIDs(ctx, experimentIDs)
	if err != nil {
		return nil, err
	}

	trialIDToWorkspaceIDs, err := db.TrialIDsToWorkspaceIDs(ctx, trialIDs)
	if err != nil {
		return nil, err
	}

	workspaceIDs := map[model.AccessScopeID]bool{}
	var workspaceIDList []model.AccessScopeID
	for _, wID := range expIDsToWorkspaceIDs {
		workspaceIDs[wID] = true
	}
	for _, wID := range trialIDToWorkspaceIDs {
		workspaceIDs[wID] = true
	}
	for wID := range workspaceIDs {
		workspaceIDList = append(workspaceIDList, wID)
	}

	return workspaceIDList, nil
}

func init() {
	AuthZProvider.Register("rbac", &NSCAuthZRBAC{})
}
