import { atomFamily, selector, selectorFamily, SetterOrUpdater, useRecoilState } from 'recoil';

import { getUserWebSetting, updateUserWebSetting } from 'services/api';

export const UserWebSettingsDomain = {
  Global: 'global',
  ProjectDetail: 'projectDetail',
} as const;

export const GlobalKeys = {
  Global_Theme: 'theme',
} as const;

export const ProjectDetailKeys = {
  Columns: 'columns',
  ColumnWidths: 'columnWidths',
  Each: 'each',
  TableLimit: 'tableLimit',
} as const;

type GlobalKeys = typeof GlobalKeys[keyof typeof GlobalKeys];
type ProjectDetailKeys = typeof ProjectDetailKeys[keyof typeof ProjectDetailKeys];
type UserWebSettingsSubDomainName = GlobalKeys | ProjectDetailKeys;

type UserWebSettingsDomainName = typeof UserWebSettingsDomain[keyof typeof UserWebSettingsDomain];

export type ProjectDetailData = {
  [ProjectDetailKeys.ColumnWidths]: number[];
  [ProjectDetailKeys.Columns]: string[];
  [ProjectDetailKeys.Each]: Record<
    number,
    {
      archived: boolean;
      pinned: number[];
      sortKey: string;
      userFilter: number[];
    }
  >;
  [ProjectDetailKeys.TableLimit]: number;
};

export type GlobalData = {
  [GlobalKeys.Global_Theme]: 'dark' | 'light' | 'system';
};

export type UserWebSettings = {
  [UserWebSettingsDomain.Global]: GlobalData;
  [UserWebSettingsDomain.ProjectDetail]: ProjectDetailData;
};

const defaultUserWebSettings: UserWebSettings = {
  [UserWebSettingsDomain.Global]: { [GlobalKeys.Global_Theme]: 'system' },
  [UserWebSettingsDomain.ProjectDetail]: {
    [ProjectDetailKeys.Columns]: ['id', 'name'],
    [ProjectDetailKeys.ColumnWidths]: [],
    [ProjectDetailKeys.Each]: {
      1: {
        archived: false,
        pinned: [],
        sortKey: '',
        userFilter: [],
      },
    },
    [ProjectDetailKeys.TableLimit]: 20,
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

type Param<T extends keyof UserWebSettings, S extends UserWebSettingsSubDomainName> = {
  domain: T;
  subDomain: S;
};

const userSettingsDomainState = atomFamily<
  any,
  Param<UserWebSettingsDomainName, UserWebSettingsSubDomainName>
>({
  default: selectorFamily<unknown, Param<UserWebSettingsDomainName, UserWebSettingsSubDomainName>>({
    get:
      (param) =>
      ({ get }) => {
        const domains = get(allUserSettingsState);
        const domain = domains[param.domain];
        switch (param.domain) {
          case UserWebSettingsDomain.ProjectDetail:
            return (domain as ProjectDetailData)[param.subDomain as ProjectDetailKeys];
          case UserWebSettingsDomain.Global:
            return (domain as GlobalData)[param.subDomain as GlobalKeys];
          default:
            return (domain as ProjectDetailData)[param.subDomain as ProjectDetailKeys];
        }
      },
    key: 'userSettingsDomainQuery',
  }),
  effects: [
    ({ onSet, node }) => {
      onSet((newValue) => {
        const param: Param<UserWebSettingsDomainName, UserWebSettingsSubDomainName> = JSON.parse(
          node.key.replace('userSettingsDomainState__', ''),
        );
        updateUserWebSetting({
          setting: { value: { [param.domain]: { [param.subDomain]: newValue } } },
        });
      });
    },
  ],
  key: 'userSettingsDomainState',
});

const useWebSettings = <T extends keyof UserWebSettings, S extends keyof UserWebSettings[T]>(
  domain: T,
  subDomain: S,
): [UserWebSettings[T][S], SetterOrUpdater<UserWebSettings[T][S]>] => {
  const [userWebSettings, setUserWebSettings] = useRecoilState<UserWebSettings[T][S]>(
    userSettingsDomainState({
      domain: domain,
      subDomain: subDomain as UserWebSettingsSubDomainName,
    }),
  );

  return [userWebSettings, setUserWebSettings];
};

export default useWebSettings;
