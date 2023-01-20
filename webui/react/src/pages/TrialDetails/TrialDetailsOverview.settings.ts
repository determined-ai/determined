import { array, boolean, literal, number, string, undefined as undefinedType, union } from 'io-ts';

import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';
import { TrialWorkloadFilter } from 'types';

export interface Settings {
  filter: TrialWorkloadFilter;
  metric?: string[];
  sortDesc: boolean;
  sortKey: string;
  tableLimit: number;
  tableOffset: number;
}

const config: SettingsConfig<Settings> = {
  applicableRoutespace: 'overview',
  settings: {
    filter: {
      defaultValue: TrialWorkloadFilter.CheckpointOrValidation,
      storageKey: 'filter',
      type: union([
        literal(TrialWorkloadFilter.All),
        literal(TrialWorkloadFilter.Checkpoint),
        literal(TrialWorkloadFilter.CheckpointOrValidation),
        literal(TrialWorkloadFilter.Validation),
      ]),
    },
    metric: {
      defaultValue: undefined,
      storageKey: 'metric',
      type: union([undefinedType, array(string)]),
    },
    sortDesc: {
      defaultValue: true,
      storageKey: 'sortDesc',
      type: boolean,
    },
    sortKey: {
      defaultValue: 'batches',
      storageKey: 'sortKey',
      type: string,
    },
    tableLimit: {
      defaultValue: MINIMUM_PAGE_SIZE,
      storageKey: 'tableLimit',
      type: number,
    },
    tableOffset: {
      defaultValue: 0,
      storageKey: 'tableOffset',
      type: number,
    },
  },
  storagePath: 'trial-detail',
};

export default config;
