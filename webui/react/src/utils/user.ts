import { DetailedUser, User } from 'types';

interface UserNameFields {
  displayName?: string;
  username?: string;
}

export function getDisplayName (user: DetailedUser | User | UserNameFields | undefined): string {
  return user?.displayName || user?.username || 'Unavailable';
}
