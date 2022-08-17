import { InteractiveTableSettings } from 'components/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { DEFAULT_POOL_TAB_KEY } from 'pages/ResourcepoolDetail';
import { Determinedjobv1State } from 'services/api-ts-sdk';

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
  status: 106,
  submissionTime: 117,
  type: 75,
  user: 85,
  weight: 107,
};

export interface Settings extends InteractiveTableSettings {
  sortKey: 'jobsAhead';
}

const routeSpaceForState = (jobState: Determinedjobv1State): string => {
  if (jobState === Determinedjobv1State.QUEUED) return '/queued';
  if (jobState === Determinedjobv1State.SCHEDULED) return '/active';
  return `/${DEFAULT_POOL_TAB_KEY}`;
};

const config = (jobState: Determinedjobv1State): SettingsConfig => ({
  applicableRoutespace: routeSpaceForState(jobState),
  settings: [
    {
      defaultValue: DEFAULT_COLUMNS,
      key: 'columns',
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      defaultValue: DEFAULT_COLUMNS.map((col: JobColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      key: 'columnWidths',
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: {
        baseType: BaseType.Float,
        isArray: true,
      },
    },
    {
      defaultValue: false,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: 'jobsAhead',
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
  storagePath: `job-queue-${jobState}`,
});

export default config;
