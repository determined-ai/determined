import { DetailedUser, User, UserOrGroup, UserOrGroupWithRoleInfo, UserWithRoleInfo } from 'types';

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

export function isUserWithRoleInfo(
  userOrGroup: Readonly<UserOrGroupWithRoleInfo>,
): userOrGroup is UserWithRoleInfo {
  return 'userId' in userOrGroup;
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
