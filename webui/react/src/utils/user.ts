import { V1GroupDetails, V1Group } from 'services/api-ts-sdk';
import { DetailedUser, UserOrGroupDetails, User } from 'types';

interface UserNameFields {
  displayName?: string;
  username?: string;
}

export function getDisplayName(user: DetailedUser | User | UserNameFields | undefined): string {
  return user?.displayName || user?.username || 'Unavailable';
}

export function isUser(obj: UserOrGroupDetails | V1Group | User) : string | undefined {
  const user = obj as User;
  return user?.username || user?.displayName;
}

export function getName(obj: UserOrGroupDetails | User | V1Group): string {
  const user = obj as User;
  const group = obj as V1Group;
  return isUser(obj) ? getDisplayName(user) : group.name ? group.name : '';
}
