import { BaseType, SettingsConfig } from 'hooks/useSettings';

export interface Settings {
  after?: string;
  before?: string;
  groupBy: string;
}

export enum GroupBy {
  Day = 'day',
  Month = 'month',
}

const config: SettingsConfig = {
  settings: [
    {
      key: 'after',
      type: { baseType: BaseType.String },
    },
    {
      key: 'before',
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: GroupBy.Day,
      key: 'groupBy',
      storageKey: 'groupBy',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'cluster/historical-usage',
};

export default config;
