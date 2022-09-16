import { atomFamily, selector, selectorFamily } from 'recoil';

import { getUserWebSetting, updateUserWebSetting } from 'services/api';

export enum ProjectDetailKey {
  Archived = 'archived',
  ColumnWidths = 'columnWidths',
  Pinned = 'pinned',
  SortKey = 'sortKey',
  TableLimit = 'tableLimit',
  UserFilter = 'userFilter',
}

export enum ThemeKey {
  Theme = 'theme'
}

export type ProjectDetail = {
  [ProjectDetailKey.Archived]: { archived: boolean };
  [ProjectDetailKey.ColumnWidths]: {columnWidths: number[]};
  [ProjectDetailKey.Pinned]: {pinned: Record<number, number[]>};
  [ProjectDetailKey.SortKey]: {sortKey: string};
  [ProjectDetailKey.TableLimit]: {tableLimit: number};
  [ProjectDetailKey.UserFilter]: {userFilter: number[]};
};

export type Theme = {
  [ThemeKey.Theme]: {theme: 'dark' | 'light' | 'system'};
};

type ProjectDetailDB = {
  [ProjectDetailKey.Archived]:boolean;
  [ProjectDetailKey.ColumnWidths]: number[];
  [ProjectDetailKey.Pinned]: Record<number, number[]>;
  [ProjectDetailKey.SortKey]: string;
  [ProjectDetailKey.TableLimit]: number;
  [ProjectDetailKey.UserFilter]: number[];
};

type ThemeDB = {
  [ThemeKey.Theme]: 'dark' | 'light' | 'system';
};

export type AllData = ProjectDetail & Theme;

export type AllDataDB = ProjectDetailDB & ThemeDB;

const getDefaultAllData = (newSettings: AllDataDB): AllData => {
  return {
    [ProjectDetailKey.Archived]: { archived: newSettings.archived ?? false },
    [ProjectDetailKey.ColumnWidths]: { columnWidths: newSettings.columnWidths ?? [] },
    [ProjectDetailKey.Pinned]: { pinned: newSettings.pinned ?? { 1: [] } },
    [ProjectDetailKey.SortKey]: { sortKey: newSettings.sortKey ?? '' },
    [ProjectDetailKey.TableLimit]: { tableLimit: newSettings.tableLimit ?? 20 },
    [ProjectDetailKey.UserFilter]: { userFilter: newSettings.userFilter ?? [] },
    [ThemeKey.Theme]: { theme: newSettings.theme ?? 'system' },
  };
};

type DomainName = ProjectDetailKey | ThemeKey;

const allUserSettings = selector<AllData>({
  get: async () => {
    try {
      const response = await getUserWebSetting({});
      const settings: AllData = getDefaultAllData(response.settings);
      return settings;
    } catch (e) {
      throw new Error('d');
    }
  },
  key: 'allUserSettings',
});

// const userSettingsDomainsState = atom<Set<string>>({
//   default: new Set(Object.keys(DomainName)), // TODO: add more
//   key: 'userSettingsDomainsState',
// });

const getDomain = <K extends keyof AllData>(
  domains: AllData,
  domainName: K,
): typeof domains[K] => {
  return domains[domainName];
};

export const userSettingsDomainState = atomFamily<any, DomainName>({
  default: selectorFamily<unknown, DomainName>({
    get: (domain) => ({ get }) => {
      const domains = get(allUserSettings);
      return getDomain(domains, domain);
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
