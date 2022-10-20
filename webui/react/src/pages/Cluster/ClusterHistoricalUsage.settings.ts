import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { ValueOf } from 'shared/types';

export interface Settings {
  after?: string;
  before?: string;
  groupBy: string;
}

export const GroupBy = {
  Day: 'day',
  Month: 'month',
} as const;

export type GroupBy = ValueOf<typeof GroupBy>;

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
