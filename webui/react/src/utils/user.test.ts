import { DetailedUser } from 'types';

import { getDisplayName } from './user';

const DISPLAY_NAME_VALUE = 'Display Name';
const USERNAME_VALUE = 'username_1';
const DEFAULT_VALUE = 'Unavailable';

const users: Array<DetailedUser> = [
  {
    displayName: DISPLAY_NAME_VALUE,
    id: 1,
    isActive: true,
    isAdmin: false,
    username: USERNAME_VALUE,
  },
  {
    id: 1,
    isActive: true,
    isAdmin: false,
    username: USERNAME_VALUE,
  },
  {
    displayName: '',
    id: 1,
    isActive: true,
    isAdmin: false,
    username: USERNAME_VALUE,
  },
  {
    id: 1,
    isActive: true,
    isAdmin: false,
    username: '',
  },
];

describe('getDisplayName', () => {
  it('returns a display name if display name is valid', () => {
    expect(getDisplayName(users[0])).toBe(DISPLAY_NAME_VALUE);
  });

  it('returns a username if display name does not exist', () => {
    expect(getDisplayName(users[1])).toBe(USERNAME_VALUE);
  });

  it('returns a username if display name has no length', () => {
    expect(getDisplayName(users[2])).toBe(USERNAME_VALUE);
  });

  it('returns a string if display name does not exist and username has no length', () => {
    expect(getDisplayName(users[3])).toBe(DEFAULT_VALUE);
  });
});
