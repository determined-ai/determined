import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetExperimentTrialsRequestSortBy } from 'services/api-ts-sdk';
import { RunState } from 'types';

export type TrialColumnName =
  | 'action'
  | 'id'
  | 'state'
  | 'totalBatchesProcessed'
  | 'bestValidationMetric'
  | 'latestValidationMetric'
  | 'startTime'
  | 'duration'
  | 'autoRestarts'
  | 'checkpoint';

export const DEFAULT_COLUMNS: TrialColumnName[] = [
  'id',
  'state',
  'totalBatchesProcessed',
  'bestValidationMetric',
  'latestValidationMetric',
  'startTime',
  'duration',
  'autoRestarts',
  'checkpoint',
];

export const DEFAULT_COLUMN_WIDTHS: Record<TrialColumnName, number> = {
  action: 46,
  autoRestarts: 117,
  bestValidationMetric: 150,
  checkpoint: 150,
  duration: 117,
  id: 85,
  latestValidationMetric: 150,
  startTime: 117,
  state: 64,
  totalBatchesProcessed: 74,
};

export const isOfSortKey = (sortKey: React.Key): sortKey is V1GetExperimentTrialsRequestSortBy => {
  return Object.values(V1GetExperimentTrialsRequestSortBy).includes(String(sortKey));
};

export interface Settings extends InteractiveTableSettings {
  columns: TrialColumnName[];
  compare: boolean;
  row?: number[];
  sortDesc: boolean;
  sortKey: V1GetExperimentTrialsRequestSortBy;
  state?: RunState[];
  tableLimit: number;
  tableOffset: number;
}

const config: SettingsConfig = {
  applicableRoutespace: '/trials',
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
      defaultValue: DEFAULT_COLUMNS.map((col: TrialColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
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
      key: 'compare',
      type: { baseType: BaseType.Boolean },
    },
    {
      key: 'row',
      type: { baseType: BaseType.Integer, isArray: true },
    },
    {
      defaultValue: true,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: V1GetExperimentTrialsRequestSortBy.ID,
      key: 'sortKey',
      storageKey: 'sortKey',
      type: { baseType: BaseType.String },
    },
    {
      key: 'state',
      storageKey: 'state',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
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
  storagePath: 'experiment-trials-list',
};

export default config;
