import { literal, string, undefined as undefinedType, union } from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';
import { ValueOf } from 'types';

export interface Settings {
  from?: string;
  to?: string;
  groupBy: GroupBy;
}

export const GroupBy = {
  Day: 'day',
  Month: 'month',
} as const;

export type GroupBy = ValueOf<typeof GroupBy>;

const config: SettingsConfig<Settings> = {
  settings: {
    from: {
      defaultValue: undefined,
      storageKey: 'from',
      type: union([undefinedType, string]),
    },
    groupBy: {
      defaultValue: GroupBy.Day,
      storageKey: 'groupBy',
      type: union([literal(GroupBy.Day), literal(GroupBy.Month)]),
    },
    to: {
      defaultValue: undefined,
      storageKey: 'to',
      type: union([undefinedType, string]),
    },
  },
  storagePath: 'cluster-historical-usage',
};

export default config;
