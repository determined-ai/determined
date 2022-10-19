import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { CommandState, CommandType } from 'types';

export type TaskColumnName =
  | 'action'
  | 'id'
  | 'startTime'
  | 'state'
  | 'name'
  | 'type'
  | 'resourcePool'
  | 'user';

export const DEFAULT_COLUMNS: TaskColumnName[] = [
  'id',
  'type',
  'name',
  'startTime',
  'state',
  'resourcePool',
  'user',
];

export const DEFAULT_COLUMN_WIDTHS: Record<TaskColumnName, number> = {
  action: 46,
  id: 100,
  name: 150,
  resourcePool: 128,
  startTime: 117,
  state: 106,
  type: 85,
  user: 85,
};

export const ALL_SORTKEY = [
  'id',
  'name',
  'resourcePool',
  'startTime',
  'state',
  'type',
  'user',
] as const;

type SORTKEYTuple = typeof ALL_SORTKEY;

export type SORTKEY = SORTKEYTuple[number];

export const isOfSortKey = (sortKey: React.Key): sortKey is SORTKEY => {
  return !!ALL_SORTKEY.find((d) => d === String(sortKey));
};

export interface Settings extends InteractiveTableSettings {
  columns: TaskColumnName[];
  row?: string[];
  search?: string;
  sortKey: SORTKEY;
  state?: CommandState[];
  type?: CommandType[];
  user?: string[];
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
      defaultValue: DEFAULT_COLUMNS.map((col: TaskColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
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
      defaultValue: 'startTime',
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
  storagePath: 'task-list',
};

export default config;
