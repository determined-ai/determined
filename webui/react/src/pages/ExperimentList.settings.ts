// @ts-nocheck
import { MINIMUM_PAGE_SIZE } from 'components/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { RunState } from 'types';

export const DEFAULT_COLUMNS = [
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

export const DEFAULT_COLUMN_WIDTHS = {
  description: 139,
  duration: 63,
  forkedFrom: 83,
  id: 41,
  name: 122,
  progress: 104,
  resourcePool: 88,
  searcherType: 122,
  startTime: 86,
  state: 82,
  tags: 80,
  trials: 49,
  user: 80,
};

export interface Settings {
  archived?: boolean;
  columns?: string[];
  label?: string[];
  row?: number[];
  search?: string;
  sortDesc: boolean;
  sortKey: V1GetExperimentsRequestSortBy;
  state?: RunState[];
  tableLimit: number;
  tableOffset: number;
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
      defaultValue: DEFAULT_COLUMNS.map(col => DEFAULT_COLUMN_WIDTHS[col]),
      key: 'columnWidths',
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
