import { atomFamily, selector, selectorFamily, SetterOrUpdater, useRecoilState } from 'recoil';

import { getUserWebSetting, updateUserWebSetting } from 'services/api';

export const ProjectDetailType = {
  Archived: 'pd_archived',
  ColumnWidths: 'pd_columnWidths',
  Pinned: 'pd_pinned',
  SortKey: 'pd_sortKey',
  TableLimit: 'pd_tableLimit',
  UserFilter: 'pd_userFilter',
} as const;

type ProjectDetailType = typeof ProjectDetailType[keyof typeof ProjectDetailType];

export const ThemeType = { Theme: 'theme' } as const;

type ThemeType = typeof ThemeType[keyof typeof ThemeType];

export type ProjectDetail = {
  [ProjectDetailType.Archived]: boolean;
  [ProjectDetailType.ColumnWidths]: number[];
  [ProjectDetailType.Pinned]: Record<number, number[]>;
  [ProjectDetailType.SortKey]: string;
  [ProjectDetailType.TableLimit]: number;
  [ProjectDetailType.UserFilter]: number[];
};

export type Theme = {
  [ThemeType.Theme]: 'dark' | 'light' | 'system';
};

export type AllData = ProjectDetail & Theme;

const defaultAllData: AllData = {
  [ProjectDetailType.Archived]: false,
  [ProjectDetailType.ColumnWidths]: [],
  [ProjectDetailType.Pinned]: { 1: [] },
  [ProjectDetailType.SortKey]: '',
  [ProjectDetailType.TableLimit]: 20,
  [ProjectDetailType.UserFilter]: [],
  [ThemeType.Theme]: 'system',
};

type DomainName = ProjectDetailType | ThemeType;

const allUserSettingsState = selector<AllData>({
  get: async () => {
    try {
      const response = await getUserWebSetting({});
      const resSettings = response.settings;
      Object.keys(resSettings).forEach((key: string) => {
        resSettings[key] = { [key]: resSettings[key] };
      });
      const settings: AllData = { ...defaultAllData, ...resSettings };
      return settings;
    } catch (e) {
      throw new Error('d');
    }
  },
  key: 'allUserSettingsState',
});

const userSettingsDomainState = atomFamily<any, DomainName>({
  default: selectorFamily<unknown, DomainName>({
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

const useWebSettings = <T extends DomainName>(domain: T):
[Record<T, AllData[T]>, SetterOrUpdater<Record<T, AllData[T]>>] => {
  const [ userWebSettings, setUserWebSettings ] = useRecoilState<Record<T, AllData[T]>>(
    userSettingsDomainState(domain),
  );

  return [ userWebSettings, setUserWebSettings ];
};

export default useWebSettings;
