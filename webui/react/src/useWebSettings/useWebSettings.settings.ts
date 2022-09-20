export const UserWebSettingsDomain = {
  Global: 'global',
  ProjectDetail: 'projectDetail',
} as const;

const GlobalKeys = {
  Global_Theme: 'theme',
} as const;

export const ProjectDetailKeys = {
  Columns: 'columns',
  ColumnWidths: 'columnWidths',
  Each: 'each',
  TableLimit: 'tableLimit',
} as const;

export type GlobalKeys = typeof GlobalKeys[keyof typeof GlobalKeys];
export type ProjectDetailKeys = typeof ProjectDetailKeys[keyof typeof ProjectDetailKeys];

export type UserWebSettingsSubDomainName = GlobalKeys | ProjectDetailKeys;

export type UserWebSettingsDomainName =
  typeof UserWebSettingsDomain[keyof typeof UserWebSettingsDomain];

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

export type Param<T extends keyof UserWebSettings, S extends UserWebSettingsSubDomainName> = {
  domain: T;
  subDomain: S;
};

export const defaultUserWebSettings: UserWebSettings = {
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
