import { V1Group, V1RoleWithAssignments } from 'services/api-ts-sdk';
import {
  DetailedUser,
  GroupWithRoleInfo,
  User,
  UserOrGroup,
  UserOrGroupWithRoleInfo,
  UserWithRoleInfo,
} from 'types';

interface UserNameFields {
  displayName?: string;
  username?: string;
}

export function getDisplayName(
  user: Readonly<DetailedUser | User | UserNameFields | undefined>,
): string {
  return user?.displayName || user?.username || 'Unavailable';
}

export function isUser(userOrGroup: Readonly<UserOrGroup>): userOrGroup is Readonly<User> {
  return 'username' in userOrGroup || 'displayName' in userOrGroup;
}

export function isUserWithRoleInfo(
  userOrGroup: Readonly<UserOrGroupWithRoleInfo>,
): userOrGroup is Readonly<UserWithRoleInfo> {
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
): Readonly<UserOrGroupWithRoleInfo[]> => {
  const groupsAndUsers: [
    V1RoleWithAssignments['groupRoleAssignments'],
    V1RoleWithAssignments['userRoleAssignments'],
  ][] = assignments.map((assignment: V1RoleWithAssignments) => {
    return [assignment.groupRoleAssignments, assignment.userRoleAssignments];
  });
  const groups: GroupWithRoleInfo[] = groupsAndUsers
    .flatMap((data) => data?.[0] ?? [])
    .map((d) => {
      const group = groupsAssignedDirectly.find((g) => g.groupId === d.groupId);
      const groupWithRole: GroupWithRoleInfo = {
        groupId: group?.groupId,
        groupName: group?.name,
        roleAssignment: d.roleAssignment,
      };
      return groupWithRole;
    })
    .filter((d) => d.groupId);
  const users: UserWithRoleInfo[] = groupsAndUsers
    .flatMap((data) => data?.[1] ?? [])
    .map((d) => {
      const user = usersAssignedDirectly.find((u) => u.id === d.userId);
      const userWithRole: UserWithRoleInfo = {
        displayName: user?.displayName,
        roleAssignment: d.roleAssignment,
        userId: user?.id ?? -1,
        username: user?.username ?? '',
      };
      return userWithRole;
    })
    .filter((d) => d.userId !== -1);
  return [...groups, ...users];
};
