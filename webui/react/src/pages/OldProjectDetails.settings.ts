import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
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
  archived: 80,
  description: 148,
  duration: 96,
  forkedFrom: 128,
  id: 60,
  name: 150,
  numTrials: 74,
  progress: 111,
  resourcePool: 140,
  searcherType: 140,
  startTime: 118,
  state: 64,
  tags: 106,
  user: 85,
};

export interface ProjectDetailsSettings extends InteractiveTableSettings {
  archived?: boolean;
  columns: ExperimentColumnName[];
  label?: string[];
  pinned: Record<number, number[]>; // key is `projectId`, value is array of experimentId
  row?: number[];
  search?: string;
  sortKey: V1GetExperimentsRequestSortBy;
  state?: RunState[];
  user?: string[];
}

const config: SettingsConfig = {
  applicableRoutespace: '/projects',
  settings: [
    {
      defaultValue: { 1: [] },
      key: 'pinned',
      skipUrlEncoding: true,
      storageKey: 'pinned',
      type: { baseType: BaseType.Object },
    },
    {
      defaultValue: false,
      key: 'archived',
      storageKey: 'archived',
      type: { baseType: BaseType.Boolean },
    },
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
      storageKey: 'row',
      type: { baseType: BaseType.Integer, isArray: true },
    },
    {
      key: 'search',
      storageKey: 'search',
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
      storageKey: 'tableOffset',
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
  storagePath: 'project-details',
};

export default config;
