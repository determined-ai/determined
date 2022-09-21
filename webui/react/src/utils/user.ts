import { V1GroupDetails } from 'services/api-ts-sdk';
import { DetailedUser, UserOrGroupDetails, User } from 'types';

interface UserNameFields {
  displayName?: string;
  username?: string;
}

export function getDisplayName(user: DetailedUser | User | UserNameFields | undefined): string {
  return user?.displayName || user?.username || 'Unavailable';
}

export function isUser(obj: UserOrGroupDetails): string | undefined {
  const user = obj as User;
  return user?.username || user?.displayName;
}

export function getName(obj: UserOrGroupDetails): string {
  const user = obj as User;
  const group = obj as V1GroupDetails;
  return isUser(obj) ? getDisplayName(user) : group.name ? group.name : '';
}
