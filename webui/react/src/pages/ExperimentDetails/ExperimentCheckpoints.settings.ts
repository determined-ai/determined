import { InteractiveTableSettings } from 'components/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetExperimentCheckpointsRequestSortBy } from 'services/api-ts-sdk';
import { CheckpointState } from 'types';

export type CheckpointColumnName =
  | 'action'
  | 'uuid'
  | 'state'
  | 'searcherMetric'
  | 'totalBatches'
  | 'checkpoint';

export const DEFAULT_COLUMNS: CheckpointColumnName[] = [
  'uuid',
  'state',
  'totalBatches',
  'searcherMetric',
  'checkpoint',
];

export const DEFAULT_COLUMN_WIDTHS: Record<CheckpointColumnName, number> = {
  action: 46,
  checkpoint: 100,
  searcherMetric: 100,
  state: 117,
  totalBatches: 74,
  uuid: 200,
};

export interface Settings extends InteractiveTableSettings {
  columns: CheckpointColumnName[];
  row?: string[];
  sortDesc: boolean;
  sortKey: V1GetExperimentCheckpointsRequestSortBy;
  state?: CheckpointState[];
  tableLimit: number;
  tableOffset: number;
}

const config: SettingsConfig = {
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
      defaultValue: DEFAULT_COLUMNS.map((col: CheckpointColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      key: 'columnWidths',
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: {
        baseType: BaseType.Float,
        isArray: true,
      },
    },
    {
      key: 'row',
      type: { baseType: BaseType.String, isArray: true },
    },
    {
      defaultValue: true,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: V1GetExperimentCheckpointsRequestSortBy.UUID,
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
  storagePath: 'experiment-checkpoints-list',
};

export default config;
