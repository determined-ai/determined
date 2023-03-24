import { array, boolean, number, string } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';
import { Jobv1State } from 'services/api-ts-sdk';

export type JobColumnName =
  | 'action'
  | 'preemptible'
  | 'type'
  | 'submissionTime'
  | 'name'
  | 'status'
  | 'slots'
  | 'priority'
  | 'weight'
  | 'resourcePool'
  | 'user';

export const DEFAULT_COLUMNS: JobColumnName[] = [
  'preemptible',
  'type',
  'name',
  'priority',
  'submissionTime',
  'slots',
  'status',
  'user',
];

export const DEFAULT_COLUMN_WIDTHS: Record<JobColumnName, number> = {
  action: 46,
  name: 150,
  preemptible: 106,
  priority: 107,
  resourcePool: 107,
  slots: 74,
  status: 160,
  submissionTime: 117,
  type: 75,
  user: 85,
  weight: 107,
};

export interface Settings extends InteractiveTableSettings {
  sortKey: string;
}

const config = (jobState: Jobv1State): SettingsConfig<Settings> => ({
  settings: {
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(string),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: JobColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: array(number),
    },
    sortDesc: {
      defaultValue: false,
      storageKey: 'sortDesc',
      type: boolean,
    },
    sortKey: {
      defaultValue: 'jobsAhead',
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
  storagePath: `job-queue-${jobState}`,
});

export default config;
