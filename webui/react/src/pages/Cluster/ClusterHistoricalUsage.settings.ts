import { literal, string, undefined as undefinedType, union } from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';
import { ValueOf } from 'types';

export interface Settings {
  after?: string;
  before?: string;
  groupBy: GroupBy;
}

export const GroupBy = {
  Day: 'day',
  Month: 'month',
} as const;

export type GroupBy = ValueOf<typeof GroupBy>;

const config: SettingsConfig<Settings> = {
  settings: {
    after: {
      defaultValue: undefined,
      storageKey: 'after',
      type: union([undefinedType, string]),
    },
    before: {
      defaultValue: undefined,
      storageKey: 'before',
      type: union([undefinedType, string]),
    },
    groupBy: {
      defaultValue: GroupBy.Day,
      storageKey: 'groupBy',
      type: union([literal(GroupBy.Day), literal(GroupBy.Month)]),
    },
  },
  storagePath: 'cluster-historical-usage',
};

export default config;
