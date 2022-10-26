import { number, string } from '@recoiljs/refine';
import { atom, atomFamily, selector, selectorFamily } from 'recoil';
import { urlSyncEffect } from 'recoil-sync';

import { getUserWebSetting, updateUserWebSetting } from 'services/api';

export const UserWebSettingsDomain = {
  ProjectDetail: 'projectDetail',
} as const;

export const ProjectDetailKeys = {
  Columns: 'columns',
  ColumnWidths: 'columnWidths',
  TableLimit: 'tableLimit',
} as const;

export type ProjectDetailKeys = typeof ProjectDetailKeys[keyof typeof ProjectDetailKeys];

export type UserWebSettingsSubDomainName = ProjectDetailKeys;

export type UserWebSettingsDomainName =
  typeof UserWebSettingsDomain[keyof typeof UserWebSettingsDomain];

export type ProjectDetailData = {
  [ProjectDetailKeys.ColumnWidths]: number[];
  [ProjectDetailKeys.Columns]: string[];
  [ProjectDetailKeys.TableLimit]: number;
  [projectId: number]: {
    archived: boolean;
    pinned: number[];
    sortKey: string;
    userFilter: number[];
  };
};

export type UserWebSettings = {
  [UserWebSettingsDomain.ProjectDetail]: ProjectDetailData;
};

const defaultUserWebSettings: UserWebSettings = {
  [UserWebSettingsDomain.ProjectDetail]: {
    [ProjectDetailKeys.Columns]: ['id', 'name'],
    [ProjectDetailKeys.ColumnWidths]: [],
    [ProjectDetailKeys.TableLimit]: 20,
    1: {
      archived: false,
      pinned: [],
      sortKey: '',
      userFilter: [],
    },
  },
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

const userSettingsSelector = selectorFamily<unknown, UserWebSettingsSubDomainName>({
  get:
    (subDomain) =>
    ({ get }) => {
      const domains = get(allUserSettingsState);
      const domain = domains['projectDetail'];
      return domain[subDomain];
    },
  key: 'userSettingsDomainQuery',
});

const userSettingsDomainState = atomFamily<any, UserWebSettingsSubDomainName>({
  default: userSettingsSelector,
  effects: (subDomain) => [
    ({ onSet }) => {
      onSet((newValue) => {
        updateUserWebSetting({
          setting: { value: { projectDetail: { [subDomain]: newValue } } },
        });
      });
    },
  ],
  key: 'userSettingsDomainState',
});

export const config = {
  settings: {
    columns: {
      atom: userSettingsDomainState('columns'),
    },
    columnWidths: {
      atom: userSettingsDomainState('columnWidths'),
    },
    letter: {
      atom: atom<string>({
        default: 'abc',
        effects: [
          urlSyncEffect({ history: 'replace', refine: string(), storeKey: 'projectDetail' }),
        ],
        key: 'letter',
      }),
    },
    numOfCake: {
      atom: atom<number>({
        default: 0,
        effects: [
          urlSyncEffect({
            history: 'replace',
            refine: number(),
            storeKey: 'projectDetail',
          }),
        ],
        key: 'numOfCake',
      }),
    },
    tableLimit: {
      atom: userSettingsDomainState('tableLimit'),
    },
  },
} as const;
