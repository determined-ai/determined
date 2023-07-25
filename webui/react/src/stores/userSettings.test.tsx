import * as t from 'io-ts';

import authStore from 'stores/auth';
import userStore from 'stores/users';

import { UserSettingsStore } from './userSettings';

const CURRENT_USER = { id: 1, isActive: true, isAdmin: false, username: 'bunny' };

vi.mock('services/api', () => ({
  getUserSetting: () => Promise.resolve({ settings: [] }),
  updateUserSetting: () => Promise.resolve(),
}));

const Config = t.type({
  boolean: t.boolean,
  booleanArray: t.union([t.array(t.boolean), t.undefined]),
  number: t.union([t.undefined, t.number]),
  numberArray: t.array(t.number),
  string: t.union([t.undefined, t.string]),
  stringArray: t.union([t.undefined, t.array(t.string)]),
});
const configPath = 'settings-normal';

const setup = () => {
  authStore.setAuth({ isAuthenticated: true });
  authStore.setAuthChecked();
  userStore.updateCurrentUser(CURRENT_USER);
  return new UserSettingsStore();
};

describe('userSettings', () => {
  const newSettings = {
    boolean: false,
    booleanArray: [false, true],
    number: 3.14e-12,
    numberArray: [0, 100, -5280],
    string: 'Hello World',
    stringArray: ['abc', 'def', 'ghi'],
  };

  afterEach(() => vi.clearAllMocks());

  it('should update settings', async () => {
    const store = setup();

    const p = store
      .get(Config, configPath)
      .toPromise()
      .then((n) => expect(n).toStrictEqual(newSettings));
    store.set(Config, configPath, newSettings);
    await p;
  });
});
