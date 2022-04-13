import { InteractiveTableSettings } from 'components/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { RunState } from 'types';

export type ExperimentColumnName =
  | 'action'
  | 'archived'
  | 'description'
  | 'duration'
  | 'forkedFrom'
  | 'id'
  | 'name'
  | 'progress'
  | 'resourcePool'
  | 'searcherType'
  | 'startTime'
  | 'state'
  | 'tags'
  | 'numTrials'
  | 'user';

export const DEFAULT_COLUMNS: ExperimentColumnName[] = [
  'id',
  'name',
  'description',
  'tags',
  'forkedFrom',
  'startTime',
  'state',
  'searcherType',
  'user',
];

export const DEFAULT_COLUMN_WIDTHS: Record<ExperimentColumnName, number> = {
  action: 46,
  archived: 75,
  description: 147,
  duration: 96,
  forkedFrom: 128,
  id: 57,
  name: 150,
  numTrials: 74,
  progress: 111,
  resourcePool: 128,
  searcherType: 129,
  startTime: 117,
  state: 106,
  tags: 106,
  user: 85,
};

export interface ExperimentListSettings extends InteractiveTableSettings {
  archived?: boolean;
  columns: ExperimentColumnName[];
  label?: string[];
  search?: string;
  sortKey: V1GetExperimentsRequestSortBy;
  state?: RunState[];
  user?: string[];
}

const config: SettingsConfig = {
  settings: [
    {
      defaultValue: false,
      key: 'archived',
      storageKey: 'archived',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: DEFAULT_COLUMNS,
      key: 'columns',
      storageKey: 'columns',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      defaultValue: DEFAULT_COLUMNS.map((col: ExperimentColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      key: 'columnWidths',
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: {
        baseType: BaseType.Float,
        isArray: true,
      },

    },
    {
      key: 'label',
      storageKey: 'label',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'row',
      type: { baseType: BaseType.Integer, isArray: true },
    },
    {
      key: 'search',
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: true,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: V1GetExperimentsRequestSortBy.STARTTIME,
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
    {
      key: 'type',
      storageKey: 'type',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'user',
      storageKey: 'user',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
  ],
  storagePath: 'experiment-list',
};

export default config;
