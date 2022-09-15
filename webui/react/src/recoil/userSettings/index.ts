import { atomFamily, selector, selectorFamily } from 'recoil';

import { getUserWebSetting } from 'services/api';

export enum DomainName {
  ProjectDetail = 'project-detail',
  Theme = 'theme',
}

export type ProjectDetail = {
  archived: boolean;
  columnWidths: number[];
  pinned: Record<number, number[]>;
  sortKey: string;
  tableLimit: number;
  userFilter: number[];
};

export type Theme = {
  theme: 'dark' | 'light' | 'system';
};

type AllData = {
  [DomainName.ProjectDetail]: ProjectDetail;
  [DomainName.Theme]: Theme;
};

const allUserSettings = selector<AllData>({
  get: async () => {
    try {
      const response = await getUserWebSetting({});
      const a: AllData = (<AllData>{}) as AllData;
      console.log(a);
      const settings = response.settings as AllData;
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
  domainName: K
): typeof domains[K] => {
  return domains[domainName];
};

export const userSettingsDomainState = atomFamily<any, DomainName>({
  default: selectorFamily<unknown, DomainName>({
    get:
      (domain) =>
      ({ get }) => {
        const domains = get(allUserSettings);
        return getDomain(domains, domain);
      },
    key: 'userSettingsDomainQuery',
  }),
  key: 'userSettingsDomainState',
});
