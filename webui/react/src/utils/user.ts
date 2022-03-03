import { DetailedUser, User } from 'types';

export function getDisplayName (user: DetailedUser | User | undefined): string {
  return user?.displayName || user?.username || 'Unavailable';
}
