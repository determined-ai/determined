package command

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

// NSCAuthZRBAC is the RBAC implementation of the NSCAuthZ interface.
type NSCAuthZRBAC struct{}

func (a *NSCAuthZRBAC) accessibleScopes(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
	permission rbacv1.PermissionType,
) (model.AccessScopeSet, error) {
	returnScope := model.AccessScopeSet{}
	var workspaces []int

	// check if user has global permissions
	err := db.DoesPermissionMatch(ctx, curUser.ID, nil, permission)
	if err == nil {
		if requestedScope == 0 {
			err = db.Bun().NewSelect().Table("workspaces").Column("id").Scan(ctx, &workspaces)
			if err != nil {
				return nil, errors.Wrapf(err, "error getting workspaces from db")
			}

			for _, workspaceID := range workspaces {
				returnScope[model.AccessScopeID(workspaceID)] = true
			}
			return returnScope, nil
		}
		return model.AccessScopeSet{requestedScope: true}, nil
	}

	// get all workspaces user has permissions to
	workspaces, err = db.GetNonGlobalWorkspacesWithPermission(
		ctx, curUser.ID, permission)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting workspaces from db")
	}

	if requestedScope == 0 {
		for _, workspaceID := range workspaces {
			returnScope[model.AccessScopeID(workspaceID)] = true
		}
		return returnScope, nil
	}

	for _, workspaceID := range workspaces {
		if requestedScope == model.AccessScopeID(workspaceID) {
			return model.AccessScopeSet{requestedScope: true}, nil
		}
	}
	return model.AccessScopeSet{}, nil
}

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
) (permErr *authz.PermissionDeniedError, serverErr error) {
	fields := audit.ExtractLogFields(ctx)
	a.addLogInfoWorkspaces(&fields, curUser, permission, workspaceIDs)
	defer func() {
		if serverErr == nil {
			fields["permissionGranted"] = permErr == nil
			audit.Log(fields)
		}
	}()
	wids := []int32{}
	for _, id := range workspaceIDs {
		wids = append(wids, int32(id))
	}
	if err := db.DoesPermissionMatchAll(ctx, curUser.ID, permission, wids...); err != nil {
		switch typedErr := err.(type) {
		case authz.PermissionDeniedError:
			return &typedErr, nil
		default:
			return nil, err
		}
	}
	return nil, nil
}

func (a *NSCAuthZRBAC) checkForPermission(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	permission rbacv1.PermissionType,
) (permErr *authz.PermissionDeniedError, serverErr error) {
	fields := audit.ExtractLogFields(ctx)
	a.addLogInfo(&fields, curUser, permission, workspaceID)
	defer func() {
		if serverErr == nil {
			fields["permissionGranted"] = permErr == nil
			audit.Log(fields)
		}
	}()

	wID := int32(workspaceID)
	if err := db.DoesPermissionMatch(ctx, curUser.ID, &wID,
		permission); err != nil {
		switch typedErr := err.(type) {
		case authz.PermissionDeniedError:
			return &typedErr, nil
		default:
			return nil, err
		}
	}
	return nil, nil
}

// CanGetNSC checks if the user is authorized to view NSCs in the specified workspace.
func (a *NSCAuthZRBAC) CanGetNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) (canGetCmd bool, serverErr error) {
	permErr, serverErr := a.checkForPermission(ctx, curUser, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_NSC)
	return permErr == nil, serverErr
}

// CanGetActiveTasksCount always returns a nil error.
func (a *NSCAuthZRBAC) CanGetActiveTasksCount(ctx context.Context, curUser model.User) (err error) {
	return nil
}

// CanTerminateNSC checks if the user is authorized to terminate NSCs in the workspace.
func (a *NSCAuthZRBAC) CanTerminateNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) (err error) {
	permErr, serverErr := a.checkForPermission(ctx, curUser, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_NSC)
	if permErr != nil {
		return *permErr
	}
	return serverErr
}

// CanCreateNSC checks if the user is authorized to create NSCs in the workspace.
func (a *NSCAuthZRBAC) CanCreateNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	// TODO(DET-8774): the caller should check for workspace status (archived).
	permErr, serverErr := a.checkForPermission(ctx, curUser, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_NSC)
	if permErr != nil {
		return *permErr
	}
	return serverErr
}

// CanSetNSCsPriority checks if the user is authorized to set NSCs priority in the workspace.
func (a *NSCAuthZRBAC) CanSetNSCsPriority(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID, priority int,
) error {
	// CHECK(DET-8794): we only just need workspaceID here.
	permErr, serverErr := a.checkForPermission(ctx, curUser, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_NSC)
	if permErr != nil {
		return *permErr
	}
	return serverErr
}

// AccessibleScopes returns the set of scopes that the user should be limited to.
func (a *NSCAuthZRBAC) AccessibleScopes(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
) (model.AccessScopeSet, error) {
	return a.accessibleScopes(ctx, curUser, requestedScope,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_NSC)
}

// FilterTensorboards returns the tensorboards that the user has access to.
func (a *NSCAuthZRBAC) FilterTensorboards(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
	tensorboards []*tensorboardv1.Tensorboard,
) ([]*tensorboardv1.Tensorboard, error) {
	var filteredTensorboards []*tensorboardv1.Tensorboard
	filteredScopes, err := a.accessibleScopes(ctx, curUser, requestedScope,
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
		for wID := range workspaceIDs {
			if _, ok := filteredScopes[model.AccessScopeID(wID)]; !ok {
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
) (canGetTensorboards bool, serverError error) {
	workspaceIDs := []model.AccessScopeID{workspaceID}

	expToWorkspaceIDs, err := db.ExperimentIDsToWorkspaceIDs(ctx, experimentIDs)
	if err != nil {
		return false, errors.Wrapf(err, "error getting workspaceIDs from experiment IDs")
	}

	trialsToWorkspaceIDs, err := db.TrialIDsToWorkspaceIDs(ctx, trialIDs)
	if err != nil {
		return false, errors.Wrapf(err, "error getting workspaceIDs from trial IDs")
	}
	workspaceIDs = append(workspaceIDs, expToWorkspaceIDs...)
	workspaceIDs = append(workspaceIDs, trialsToWorkspaceIDs...)

	accessDenied, err := a.checkForPermissions(ctx, curUser,
		workspaceIDs, rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_ARTIFACTS)

	return accessDenied == nil, err
}

// CanTerminateTensorboard always returns nil.
func (a *NSCAuthZRBAC) CanTerminateTensorboard(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	experimentIDs []int32, trialIDs []int32,
) error {
	return a.CanTerminateNSC(ctx, curUser, workspaceID)
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

	workspaceIDs := map[int]bool{}
	var workspaceIDList []model.AccessScopeID
	for wID := range expIDsToWorkspaceIDs {
		workspaceIDs[wID] = true
	}
	for wID := range trialIDToWorkspaceIDs {
		workspaceIDs[wID] = true
	}
	for wID := range workspaceIDs {
		workspaceIDList = append(workspaceIDList, model.AccessScopeID(wID))
	}

	return workspaceIDList, nil
}

func init() {
	AuthZProvider.Register("rbac", &NSCAuthZRBAC{})
}
