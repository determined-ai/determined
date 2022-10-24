import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetModelVersionsRequestSortBy } from 'services/api-ts-sdk';

export type ModelVersionColumnName =
  | 'action'
  | 'description'
  | 'lastUpdatedTime'
  | 'name'
  | 'tags'
  | 'version'
  | 'user';

export const DEFAULT_COLUMNS: ModelVersionColumnName[] = [
  'version',
  'name',
  'description',
  'lastUpdatedTime',
  'tags',
  'user',
];

export const DEFAULT_COLUMN_WIDTHS: Record<ModelVersionColumnName, number> = {
  action: 46,
  description: 147,
  lastUpdatedTime: 117,
  name: 150,
  tags: 106,
  user: 85,
  version: 74,
};

export const isOfSortKey = (sortKey: React.Key): sortKey is V1GetModelVersionsRequestSortBy => {
  return Object.values(V1GetModelVersionsRequestSortBy).includes(String(sortKey));
};

export interface Settings extends InteractiveTableSettings {
  columns: ModelVersionColumnName[];
  sortKey: V1GetModelVersionsRequestSortBy;
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
      defaultValue: DEFAULT_COLUMNS.map(
        (col: ModelVersionColumnName) => DEFAULT_COLUMN_WIDTHS[col],
      ),
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
      defaultValue: V1GetModelVersionsRequestSortBy.VERSION,
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
  storagePath: 'model-details',
};

export default config;
