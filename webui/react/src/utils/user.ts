import { DetailedUser } from 'types';

export function getDisplayName (user: DetailedUser | undefined): string {
  return user?.displayName || user?.username || 'Unavailable';
}
