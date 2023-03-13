import { V1Group, V1RoleWithAssignments } from 'services/api-ts-sdk';
import {
  DetailedUser,
  GroupWithRoleInfo,
  User,
  UserOrGroup,
  UserOrGroupWithRoleInfo,
  UserWithRoleInfo,
} from 'types';

export interface UserNameFields {
  displayName?: string;
  username?: string;
}

export function getDisplayName(
  user: Readonly<DetailedUser | User | UserNameFields | undefined>,
): string {
  return user?.displayName || user?.username || 'Unavailable';
}

export function isUser(userOrGroup: Readonly<UserOrGroup>): userOrGroup is User {
  return 'username' in userOrGroup || 'displayName' in userOrGroup;
}

export function isUserWithRoleInfo(
  userOrGroup: Readonly<UserOrGroupWithRoleInfo>,
): userOrGroup is UserWithRoleInfo {
  return 'userId' in userOrGroup;
}

export function getName(userOrGroup: Readonly<UserOrGroup>): string {
  if (isUser(userOrGroup)) {
    return getDisplayName(userOrGroup);
  }
  return userOrGroup?.name ?? '';
}

export const getIdFromUserOrGroup = (userOrGroup: Readonly<UserOrGroup>): number => {
  if (isUser(userOrGroup)) {
    const user = userOrGroup;
    return user.id;
  }
  const group = userOrGroup;

  // The groupId should always exist
  return group.groupId || 0;
};

export const getUserOrGroupWithRoleInfo = (
  assignments: Readonly<V1RoleWithAssignments[]>,
  groupsAssignedDirectly: Readonly<V1Group[]>,
  usersAssignedDirectly: Readonly<User[]>,
): UserOrGroupWithRoleInfo[] => {
  const groupsAndUsers: [
    V1RoleWithAssignments['groupRoleAssignments'],
    V1RoleWithAssignments['userRoleAssignments'],
  ][] = assignments.map((assignment: V1RoleWithAssignments) => {
    return [assignment.groupRoleAssignments, assignment.userRoleAssignments];
  });
  const groups: GroupWithRoleInfo[] = groupsAndUsers
    .flatMap((data) => data?.[0] ?? [])
    .map((d) => {
      const groupnfo = groupsAssignedDirectly.find((g) => g.groupId === d.groupId);
      const groupWithRole: GroupWithRoleInfo = {
        groupId: groupnfo?.groupId,
        groupName: groupnfo?.name,
        roleAssignment: d.roleAssignment,
      };
      return groupWithRole;
    })
    .filter((d) => d.groupId);
  const users: UserWithRoleInfo[] = groupsAndUsers
    .flatMap((data) => data?.[1] ?? [])
    .map((d) => {
      const userInfo = usersAssignedDirectly.find((u) => u.id === d.userId);
      const groupWithRole: UserWithRoleInfo = {
        displayName: userInfo?.displayName,
        roleAssignment: d.roleAssignment,
        userId: userInfo?.id ?? -1,
        username: userInfo?.username ?? '',
      };
      return groupWithRole;
    })
    .filter((d) => d.userId !== -1);
  return [...groups, ...users];
};
