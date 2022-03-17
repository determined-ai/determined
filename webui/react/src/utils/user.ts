import { DetailedUser, User } from 'types';

interface UserFields {
  displayName: string;
  username: string;
}

export function getDisplayName (user: DetailedUser | User | UserFields | undefined): string {
  return user?.displayName || user?.username || 'Unavailable';
}
