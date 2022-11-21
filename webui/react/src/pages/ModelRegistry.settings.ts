import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';

export type ModelColumnName =
  | 'action'
  | 'archived'
  | 'description'
  | 'lastUpdatedTime'
  | 'name'
  | 'tags'
  | 'numVersions'
  | 'user';

export const DEFAULT_COLUMNS: ModelColumnName[] = [
  'name',
  'description',
  'numVersions',
  'lastUpdatedTime',
  'tags',
  'archived',
  'user',
];

export const DEFAULT_COLUMN_WIDTHS: Record<ModelColumnName, number> = {
  action: 46,
  archived: 75,
  description: 147,
  lastUpdatedTime: 117,
  name: 150,
  numVersions: 74,
  tags: 106,
  user: 85,
};
export const isOfSortKey = (sortKey: React.Key): sortKey is V1GetModelsRequestSortBy => {
  return Object.values(V1GetModelsRequestSortBy)
    .map((v) => v as string)
    .includes(String(sortKey));
};

export interface Settings extends InteractiveTableSettings {
  archived?: boolean;
  columns: ModelColumnName[];
  description?: string;
  name?: string;
  sortKey: V1GetModelsRequestSortBy;
  tags?: string[];
  users?: string[];
}

const config: SettingsConfig = {
  settings: [
    {
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
      defaultValue: DEFAULT_COLUMNS.map((col: ModelColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      key: 'columnWidths',
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: {
        baseType: BaseType.Float,
        isArray: true,
      },
    },
    {
      defaultValue: true,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: V1GetModelsRequestSortBy.CREATION_TIME,
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
    {
      key: 'users',
      storageKey: 'users',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'tags',
      storageKey: 'tags',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'name',
      storageKey: 'name',
      type: { baseType: BaseType.String },
    },
    {
      key: 'description',
      storageKey: 'description',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'model-registry',
};

export default config;
