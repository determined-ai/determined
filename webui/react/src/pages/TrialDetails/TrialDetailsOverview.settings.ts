import { MINIMUM_PAGE_SIZE } from 'components/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';

export interface Settings {
  filter: TrialWorkloadFilter;
  metric?: string[];
  sortDesc: boolean;
  sortKey: string;
  tableLimit: number;
  tableOffset: number;
}

export enum TrialWorkloadFilter {
  All = 'All',
  Checkpoint = 'Has Checkpoint',
  Validation = 'Has Validation',
  CheckpointOrValidation = 'Has Checkpoint or Validation',
}

const config: SettingsConfig = {
  settings: [
    {
      defaultValue: TrialWorkloadFilter.CheckpointOrValidation,
      key: 'filter',
      storageKey: 'filter',
      type: { baseType: BaseType.String },
    },
    {
      key: 'metric',
      storageKey: 'metric',
      type: { baseType: BaseType.String, isArray: true },
    },
    {
      defaultValue: true,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: 'batches',
      key: 'sortKey',
      storageKey: 'sortKey',
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: MINIMUM_PAGE_SIZE,
      key: 'tableLimit',
      storageKey: 'tableLimit',
      type: { baseType: BaseType.Integer },
    },
    {
      defaultValue: 0,
      key: 'tableOffset',
      type: { baseType: BaseType.Integer },
    },
  ],
  storagePath: 'trial-detail',
};

export default config;
