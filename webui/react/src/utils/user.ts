import { V1Group } from 'services/api-ts-sdk';
import { DetailedUser, User, UserOrGroup } from 'types';

interface UserNameFields {
  displayName?: string;
  username?: string;
}

export function getDisplayName(user: DetailedUser | User | UserNameFields | undefined): string {
  return user?.displayName || user?.username || 'Unavailable';
}

export function isUser(userOrGroup: UserOrGroup): string | undefined {
  const user = userOrGroup as User;
  return user?.username || user?.displayName;
}

export function getName(userOrGroup: UserOrGroup): string {
  const user = userOrGroup as User;
  const group = userOrGroup as V1Group;
  return isUser(userOrGroup) ? getDisplayName(user) : group.name ? group.name : '';
}

export const getIdFromUserOrGroup = (userOrGroup: UserOrGroup): number => {
  if (isUser(userOrGroup)) {
    const user = userOrGroup as User;
    return user.id;
  }
  const group = userOrGroup as V1Group;

  // The groupId should always exist
  return group.groupId || 0;
};
