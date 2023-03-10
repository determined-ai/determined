import { V1Role, V1RoleAssignment, V1RoleWithAssignments } from 'services/api-ts-sdk';
import { DetailedUser, User, UserOrGroup, UserRole } from 'types';

export interface UserNameFields {
  displayName?: string;
  username?: string;
}

export function getDisplayName(user: DetailedUser | User | UserNameFields | undefined): string {
  return user?.displayName || user?.username || 'Unavailable';
}

export function isUser(userOrGroup: Readonly<UserOrGroup>): userOrGroup is User {
  return 'username' in userOrGroup || 'displayName' in userOrGroup;
}

export function getName(userOrGroup: UserOrGroup): string {
  if (isUser(userOrGroup)) {
    return getDisplayName(userOrGroup);
  }
  return userOrGroup?.name ?? '';
}

export const getIdFromUserOrGroup = (userOrGroup: UserOrGroup): number => {
  if (isUser(userOrGroup)) {
    const user = userOrGroup;
    return user.id;
  }
  const group = userOrGroup;

  // The groupId should always exist
  return group.groupId || 0;
};

export const getAssignedRole = (
  record: UserOrGroup,
  assignments: V1RoleWithAssignments[],
): V1RoleAssignment | null => {
  const currentAssignment = assignments.find((aGroup) =>
    isUser(record)
      ? !!aGroup?.userRoleAssignments &&
        !!aGroup.userRoleAssignments.find((a) => a.userId === getIdFromUserOrGroup(record))
      : !!aGroup?.groupRoleAssignments &&
        !!aGroup.groupRoleAssignments.find((a) => a.groupId === getIdFromUserOrGroup(record)),
  );
  if (isUser(record) && !!record) {
    if (currentAssignment?.userRoleAssignments) {
      const myAssignment = currentAssignment.userRoleAssignments.find(
        (a) => a.userId === getIdFromUserOrGroup(record),
      );
      return myAssignment?.roleAssignment || null;
    }
  } else if (currentAssignment?.groupRoleAssignments) {
    const myAssignment = currentAssignment.groupRoleAssignments.find(
      (a) => a.groupId === getIdFromUserOrGroup(record),
    );
    return myAssignment?.roleAssignment || null;
  }
  return null;
};

export const getAssignableWorkspaceRoles = (
  roles: UserRole[],
  rolesAssignableToScope: V1Role[],
): UserRole[] => {
  const validRoleIds = new Set<number>();
  rolesAssignableToScope.forEach((role) => validRoleIds.add(role.roleId));
  return roles.filter((role) => validRoleIds.has(role.id));
};
