package rbac

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// GetPermissionSummary retrieves a list of all roles a user is assigned to along with
// what scopes that roles are assigned to.
func GetPermissionSummary(
	ctx context.Context, userID model.UserID,
) (map[*Role][]*RoleAssignment, error) {
	// Get a list of groups a user is in.
	groups, _, _, err := usergroup.SearchGroups(ctx, "", userID, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, errors.New("user has to be in at least one group")
	}
	groupIDs := make([]int32, len(groups))
	for i := range groups {
		groupIDs[i] = int32(groups[i].ID)
	}

	// Get all role assignments to all groups the user is in.
	var roleAssignments []*RoleAssignment
	if err = db.Bun().NewSelect().Model(&roleAssignments).
		Where("group_id IN (?)", bun.In(groupIDs)).
		Relation("Scope").
		Scan(ctx); err != nil {
		return nil, err
	}
	if len(roleAssignments) == 0 {
		return nil, nil
	}

	// Get unique roles and associate them to role assignments.
	roleIDsToAssignments := make(map[int][]*RoleAssignment)
	for _, r := range roleAssignments {
		roleIDsToAssignments[r.RoleID] = append(roleIDsToAssignments[r.RoleID], r)
	}
	var roles []*Role
	if err = db.Bun().NewSelect().Model(&roles).
		Where("id IN (?)", bun.In(maps.Keys(roleIDsToAssignments))).
		Relation("Permissions").
		Scan(ctx); err != nil {
		return nil, err
	}

	rolesToAssignments := make(map[*Role][]*RoleAssignment, len(roleIDsToAssignments))
	for _, r := range roles {
		rolesToAssignments[r] = roleIDsToAssignments[r.ID]
	}
	return rolesToAssignments, nil
}

// UserPermissionsForScope finds what permissions a user has on a give scope.
// Passing a workspaceID of zero signals to only check for globally-assigned roles.
func UserPermissionsForScope(ctx context.Context, uid model.UserID, workspaceID int,
) ([]Permission, error) {
	groups, _, _, err := usergroup.SearchGroups(ctx, "", uid, 0, 0)
	if err != nil {
		return nil, errors.Wrap(
			db.MatchSentinelError(err), "error finding user's group membership")
	}
	if len(groups) == 0 {
		return []Permission{}, nil
	}

	groupIDs := make([]int32, len(groups))
	for i := range groups {
		groupIDs[i] = int32(groups[i].ID)
	}

	var results []Permission

	query := db.Bun().NewSelect().Model(&results).Distinct().
		Join("INNER JOIN permission_assignments AS pa ON pa.permission_id=id").
		Join("INNER JOIN role_assignments AS ra ON ra.role_id=pa.role_id AND ra.group_id IN (?)",
			bun.In(groupIDs)).
		Join("INNER JOIN role_assignment_scopes AS ras ON ra.scope_id=ras.id")

	// If it's global-only
	if workspaceID == 0 {
		query = query.Where("ras.scope_workspace_id IS NULL")
	} else {
		query = query.Where("ras.scope_workspace_id IS NULL OR ras.scope_workspace_id=?", workspaceID)
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, errors.Wrap(
			db.MatchSentinelError(err),
			"error finding permissions for user")
	}

	return results, nil
}

// GetAllRoles pulls back a summary of all roles from the database and paginates them.
func GetAllRoles(ctx context.Context, excludeGlobalOnly bool, offset, limit int,
) ([]Role, int32, error) {
	var results []Role
	query := db.Bun().NewSelect().Model(&results).Relation("Permissions")

	if excludeGlobalOnly {
		query = query.Where(
			"NOT EXISTS (SELECT 1 FROM permission_assignments AS pa INNER JOIN permissions " +
				"AS p ON pa.permission_id = p.id WHERE pa.role_id = roles.id AND p.global_only)")
	}

	paginatedQuery := db.PaginateBun(query, "role_name", db.SortDirectionAsc, offset, limit)
	err := paginatedQuery.Scan(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(
			db.MatchSentinelError(err),
			"error retrieving roles")
	}

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(
			db.MatchSentinelError(err),
			"error retrieving count of roles")
	}

	return results, int32(count), nil
}

// GetRolesByIDs returns a set of roles and their assignments from the DB.
func GetRolesByIDs(ctx context.Context, ids ...int32,
) ([]*rbacv1.RoleWithAssignments, error) {
	var results []Role
	query := db.Bun().NewSelect().Model(&results).
		Relation("Permissions").
		Relation("RoleAssignments.Role").
		Relation("RoleAssignments.Group").
		Relation("RoleAssignments.Scope")

	if len(ids) > 0 {
		query = query.Where("id IN (?)", bun.In(ids))
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, errors.Wrap(
			db.MatchSentinelError(err),
			"error getting roles by id")
	}

	return Roles(results).Proto(), nil
}

// GetRolesAssignedToGroupsTx returns the set of roles assigned to a set of groups.
func GetRolesAssignedToGroupsTx(ctx context.Context, idb bun.IDB, ids ...int32) ([]Role, error) {
	if idb == nil {
		idb = db.Bun()
	}

	// Define a subquery for finding the ids of the roles we care about.
	query := idb.NewSelect().
		Column("role_id").
		Table("role_assignments").
		Where("group_id IN (?)", bun.In(ids))

	var roles []Role
	err := idb.NewSelect().Model(&roles).
		Relation("Permissions").
		Where("id IN (?)", query).
		Order("role_name").
		Scan(ctx)
	if err != nil {
		return nil, errors.Wrap(
			db.MatchSentinelError(err), "error looking up roles assigned to groups")
	}

	return roles, nil
}

// AddRoleAssignments adds the specified role assignments to users or groups.
func AddRoleAssignments(ctx context.Context, groups []*rbacv1.GroupRoleAssignment,
	users []*rbacv1.UserRoleAssignment,
) error {
	if len(groups)+len(users) == 0 {
		return nil
	}

	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "error starting transaction for adding role assignments")
	}

	defer func() {
		txErr := tx.Rollback()
		if txErr != nil && txErr != sql.ErrTxDone {
			logrus.WithError(txErr).Error("error rolling back transaction in adding role assignments")
		}
	}()

	if len(groups) > 0 {
		err = AddGroupAssignmentsTx(ctx, tx, groups)
		if err != nil {
			return errors.Wrap(
				db.MatchSentinelError(err), "error inserting group role assignments")
		}
	}

	if len(users) > 0 {
		var userGroups []*rbacv1.GroupRoleAssignment
		userGroups, err = GetGroupsFromUsersTx(ctx, tx, users)
		if err != nil {
			return errors.Wrap(
				db.MatchSentinelError(err), "error looking up user groups")
		}

		err = AddGroupAssignmentsTx(ctx, tx, userGroups)
		if err != nil {
			return errors.Wrap(
				db.MatchSentinelError(err), "error inserting role assignments for user groups")
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrapf(err, "error committing transaction for adding role assignments")
	}

	return nil
}

// RemoveRoleAssignments removes the specified role assignments from groups or users.
func RemoveRoleAssignments(ctx context.Context, groups []*rbacv1.GroupRoleAssignment,
	users []*rbacv1.UserRoleAssignment,
) error {
	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "Error starting transaction for removing role assignments")
	}

	defer func() {
		txErr := tx.Rollback()
		if txErr != nil && txErr != sql.ErrTxDone {
			logrus.WithError(txErr).Error("error rolling back transaction in removing role assignments")
		}
	}()

	if len(groups) > 0 {
		err = RemoveGroupAssignmentsTx(ctx, tx, groups)
		if err != nil {
			return errors.Wrap(
				db.MatchSentinelError(err), "error removing group assignments")
		}
	}
	if len(users) > 0 {
		var userGroups []*rbacv1.GroupRoleAssignment
		userGroups, err = GetGroupsFromUsersTx(ctx, tx, users)
		if err != nil {
			return errors.Wrap(
				db.MatchSentinelError(err), "error looking up user groups")
		}

		err = RemoveGroupAssignmentsTx(ctx, tx, userGroups)
		if err != nil {
			return errors.Wrap(
				db.MatchSentinelError(err), "error removing user group assignments")
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrapf(err, "error committing transaction for removing role assignments")
	}

	return nil
}

// GetGroupsFromUsersTx retrieves the group ids belonging to users while inside a transaction.
func GetGroupsFromUsersTx(ctx context.Context, idb bun.IDB, users []*rbacv1.UserRoleAssignment) (
	[]*rbacv1.GroupRoleAssignment, error,
) {
	if len(users) < 1 {
		return nil, nil
	}

	if idb == nil {
		idb = db.Bun()
	}

	var groups []*rbacv1.GroupRoleAssignment
	for _, user := range users {
		var g usergroup.Group
		err := idb.NewSelect().Model(&g).Where("user_id = ?", user.UserId).Scan(ctx)
		if err != nil {
			return nil, errors.Wrapf(db.MatchSentinelError(err),
				"Error getting group for user id %d", user.UserId)
		}
		groups = append(groups, &rbacv1.GroupRoleAssignment{
			GroupId:        int32(g.ID),
			RoleAssignment: user.RoleAssignment,
		})
	}

	return groups, nil
}

// AddGroupAssignmentsTx adds a role assignment to a group while inside a transaction.
func AddGroupAssignmentsTx(ctx context.Context, idb bun.IDB, groups []*rbacv1.GroupRoleAssignment,
) error {
	if len(groups) < 1 {
		return nil
	}

	if idb == nil {
		idb = db.Bun()
	}

	for _, group := range groups {
		s, err := getOrCreateRoleAssignmentScopeTx(ctx, idb, group.RoleAssignment)
		if err != nil {
			return errors.Wrapf(db.MatchSentinelError(err),
				"Error getting scope for group id %d", group.GroupId)
		}

		roleAssignment := RoleAssignment{
			GroupID: int(group.GroupId),
			RoleID:  int(group.RoleAssignment.Role.RoleId),
			ScopeID: s.ID,
		}

		// insert into role assignments
		_, err = idb.NewInsert().Model(&roleAssignment).Exec(ctx)
		if err != nil {
			return errors.Wrapf(db.MatchSentinelError(err),
				"Error inserting assignment for group id %d", group.GroupId)
		}
	}

	return nil
}

// RemoveGroupAssignmentsTx removes role assignments from groups while inside a transaction.
func RemoveGroupAssignmentsTx(ctx context.Context, idb bun.IDB,
	groups []*rbacv1.GroupRoleAssignment,
) error {
	if len(groups) < 1 {
		return nil
	}
	if idb == nil {
		idb = db.Bun()
	}

	for _, group := range groups {
		s, err := getOrCreateRoleAssignmentScopeTx(ctx, idb, group.RoleAssignment)
		if err != nil {
			return errors.Wrapf(db.MatchSentinelError(err),
				"Error getting scope for group id %d", group.GroupId)
		}

		res, err := idb.NewDelete().Table("role_assignments").
			Where("group_id = ?", group.GroupId).
			Where("role_id = ?", group.RoleAssignment.Role.RoleId).
			Where("scope_id = ?", s.ID).
			Exec(ctx)
		if err != nil {
			return errors.Wrapf(db.MatchSentinelError(err),
				"Error deleting assignment for group id %d", group.GroupId)
		}

		if foundErr := db.MustHaveAffectedRows(res, err); foundErr != nil {
			return errors.Wrapf(db.MatchSentinelError(foundErr),
				"Error deleting assignment for group id %d", group.GroupId)
		}
	}
	return nil
}

func getOrCreateRoleAssignmentScopeTx(ctx context.Context, idb bun.IDB,
	assignment *rbacv1.RoleAssignment,
) (RoleAssignmentScope, error) {
	if idb == nil {
		idb = db.Bun()
	}

	r := RoleAssignmentScope{}

	scopeSelect := idb.NewSelect().Model(&r)

	// Postgres unique constraints do not block duplicate null values
	// so we must check if a null scope already exists
	if assignment.ScopeWorkspaceId == nil {
		scopeSelect = scopeSelect.Where("scope_workspace_id IS NULL")
		err := scopeSelect.Scan(ctx)

		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return r, errors.Wrapf(db.MatchSentinelError(err), "Error checking for a null workspace")
		} else if err == nil {
			return r, nil
		}
	} else {
		scopeSelect = scopeSelect.Where("scope_workspace_id = ?", assignment.ScopeWorkspaceId.Value)

		r.WorkspaceID.Int32 = assignment.ScopeWorkspaceId.Value
		r.WorkspaceID.Valid = true
	}

	// Try to insert RoleAssignmentScope, do nothing if it already exists in the table
	_, err := idb.NewInsert().Model(&r).Ignore().Exec(ctx)
	if err != nil {
		return r, errors.Wrapf(db.MatchSentinelError(err), "Error creating a RoleAssignmentScope")
	}

	// Retrieve the role assignment scope from DB
	err = scopeSelect.Scan(ctx)
	if err != nil {
		return r, errors.Wrapf(db.MatchSentinelError(err), "Error getting RoleAssignmentScope %d", r.ID)
	}

	return r, nil
}

// DoesPermissionMatch checks for the existence of a permission in a workspace.
func DoesPermissionMatch(ctx context.Context, curUserID model.UserID, workspaceID *int32,
	permissionID rbacv1.PermissionType,
) error {
	query := db.Bun().NewSelect().
		Table("permission_assignments").
		Join("JOIN role_assignments ra ON permission_assignments.role_id = ra.role_id").
		Join("JOIN user_group_membership ugm ON ra.group_id = ugm.group_id").
		Join("JOIN role_assignment_scopes ras ON ra.scope_id = ras.id").
		Where("ugm.user_id = ?", curUserID).
		Where("permission_assignments.id = ?", permissionID)

	if workspaceID == nil {
		query = query.Where("ras.scope_workspace_id IS NULL")
	} else {
		query = query.Where("ras.scope_workspace_id = ? OR ras.scope_workspace_id IS NULL", *workspaceID)
	}

	exists, err := query.Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return grpcutil.ErrPermissionDenied
}
