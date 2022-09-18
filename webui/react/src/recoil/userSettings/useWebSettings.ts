import { atomFamily, selector, selectorFamily, SetterOrUpdater, useRecoilState } from 'recoil';

import { getUserWebSetting, updateUserWebSetting } from 'services/api';

export const UserWebSettingsKeys = {
  // Global
  Global_Theme: 'global_theme',

  // Project Detail
  PG_Archived: 'pd_archived',
  PG_ColumnWidths: 'pd_columnWidths',
  PG_Pinned: 'pd_pinned',
  PG_SortKey: 'pd_sortKey',
  PG_TableLimit: 'pd_tableLimit',
  PG_UserFilter: 'pd_userFilter',
} as const;

type UserWebSettingsDomainName = typeof UserWebSettingsKeys[keyof typeof UserWebSettingsKeys];

export type UserWebSettings = {
  // Global
  [UserWebSettingsKeys.Global_Theme]: 'dark' | 'light' | 'system';

  // Project Detail
  [UserWebSettingsKeys.PG_Archived]: boolean;
  [UserWebSettingsKeys.PG_ColumnWidths]: number[];
  [UserWebSettingsKeys.PG_Pinned]: Record<number, number[]>;
  [UserWebSettingsKeys.PG_SortKey]: string;
  [UserWebSettingsKeys.PG_TableLimit]: number;
  [UserWebSettingsKeys.PG_UserFilter]: number[];
}

const defaultUserWebSettings: UserWebSettings = {
  // Global
  [UserWebSettingsKeys.Global_Theme]: 'system',

  // Project Detail
  [UserWebSettingsKeys.PG_Archived]: false,
  [UserWebSettingsKeys.PG_ColumnWidths]: [],
  [UserWebSettingsKeys.PG_Pinned]: { 1: [] },
  [UserWebSettingsKeys.PG_SortKey]: '',
  [UserWebSettingsKeys.PG_TableLimit]: 20,
  [UserWebSettingsKeys.PG_UserFilter]: [],
};

const allUserSettingsState = selector<UserWebSettings>({
  get: async () => {
    try {
      const response = await getUserWebSetting({});
      const resSettings = response.settings;
      const settings: UserWebSettings = { ...defaultUserWebSettings, ...resSettings };
      return settings;
    } catch (e) {
      throw new Error('unable to fetch userWebSettings data');
    }
  },
  key: 'allUserSettingsState',
});

const userSettingsDomainState = atomFamily<any, UserWebSettingsDomainName>({
  default: selectorFamily<unknown, UserWebSettingsDomainName>({
    get: (domain) => ({ get }) => {
      const domains = get(allUserSettingsState);
      return { [domain]: domains[domain] };
    },
    key: 'userSettingsDomainQuery',
  }),
  effects: [
    ({ onSet }) => {
      onSet((newValue) => {
        updateUserWebSetting({ setting: { value: newValue } });
      });
    },
  ],
  key: 'userSettingsDomainState',
});

const useWebSettings = <T extends UserWebSettingsDomainName>(domain: T):
[Record<T, UserWebSettings[T]>, SetterOrUpdater<Record<T, UserWebSettings[T]>>] => {
  const [ userWebSettings, setUserWebSettings ] = useRecoilState<Record<T, UserWebSettings[T]>>(
    userSettingsDomainState(domain),
  );

  return [ userWebSettings, setUserWebSettings ];
};

export default useWebSettings;
