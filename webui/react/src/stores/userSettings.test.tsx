import { Map } from 'immutable';
import * as t from 'io-ts';

import authStore from 'stores/auth';
import userStore from 'stores/users';
import { Loadable, Loaded } from 'utils/loadable';

import { UserSettingsStore } from './userSettings';

const CURRENT_USER = { id: 1, isActive: true, isAdmin: false, username: 'bunny' };

vi.mock('services/api', () => ({
  getUserSetting: () => Promise.resolve({ settings: [] }),
  resetUserSetting: () => Promise.resolve(),
  updateUserSetting: () => Promise.resolve(),
}));

const Config = t.type({
  boolean: t.boolean,
  booleanArray: t.union([t.array(t.boolean), t.null]),
  number: t.union([t.null, t.number]),
  numberArray: t.array(t.number),
  string: t.union([t.null, t.string]),
  stringArray: t.union([t.null, t.array(t.string)]),
});
const configPath = 'settings-normal';

const setup = async (initialValue = {}) => {
  authStore.setAuth({ isAuthenticated: true });
  authStore.setAuthChecked();
  userStore.updateCurrentUser(CURRENT_USER);
  const store = new UserSettingsStore();
  await store.overwrite(Map(initialValue));
  return store;
};

describe('userSettings', () => {
  const expectedSettings = {
    boolean: false,
    booleanArray: [false, true],
    number: 3.14e-12,
    numberArray: [0, 100, -5280],
    string: 'Hello World',
    stringArray: ['abc', 'def', 'ghi'],
  };
  const expectedValue = 'henlo';

  afterEach(() => vi.clearAllMocks());

  describe('set', () => {
    it('interface should update settings when given an interface', async () => {
      const store = await setup();

      const oldSettings = store.get(Config, configPath).get();
      expect(oldSettings).not.toStrictEqual(Loaded(expectedSettings));
      store.set(Config, configPath, expectedSettings);
      const newSettings = store.get(Config, configPath).get();
      expect(newSettings).toStrictEqual(Loaded(expectedSettings));
    });

    it('basic should update settings when given a basic type', async () => {
      const store = await setup();

      const oldSettings = store.get(t.string, configPath).get();
      expect(oldSettings).not.toStrictEqual(Loaded(expectedValue));
      store.set(t.string, configPath, expectedValue);
      const newSettings = store.get(t.string, configPath).get();
      expect(newSettings).toStrictEqual(Loaded(expectedValue));
    });

    it('should accept partial updates', async () => {
      const store = await setup({ [configPath]: expectedSettings });

      store.set(Config, configPath, { string: expectedValue });
      const result = store.get(Config, configPath).get();
      expect(Loadable.map(result, (r) => r?.string)).toStrictEqual(Loaded(expectedValue));
    });
  });

  describe('update', () => {
    it('should update given an updater', async () => {
      const store = await setup({ [configPath]: expectedSettings });

      const spy = vi.fn((val) => ({ ...val, string: expectedValue }));
      store.update(Config, configPath, spy);
      expect(spy).toBeCalledTimes(1);
      const result = store.get(Config, configPath).get();
      expect(Loadable.map(result, (r) => r?.string)).toStrictEqual(Loaded(expectedValue));
      expect(Loadable.map(result, (r) => r?.stringArray)).toStrictEqual(
        Loaded(expectedSettings.stringArray),
      );
    });
  });
});
