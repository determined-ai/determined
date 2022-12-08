import { array, boolean, literal, number, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';
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
  return (Object.values(V1GetModelVersionsRequestSortBy) as Array<string>).includes(
    String(sortKey),
  );
};

export interface Settings extends InteractiveTableSettings {
  columns: ModelVersionColumnName[];
  sortKey: V1GetModelVersionsRequestSortBy;
}

const config: SettingsConfig<Settings> = {
  applicableRoutespace: 'model-details',
  settings: {
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(
        union([
          literal('action'),
          literal('description'),
          literal('lastUpdatedTime'),
          literal('name'),
          literal('tags'),
          literal('version'),
          literal('user'),
        ]),
      ),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map(
        (col: ModelVersionColumnName) => DEFAULT_COLUMN_WIDTHS[col],
      ),
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: array(number),
    },
    sortDesc: {
      defaultValue: true,
      storageKey: 'sortDesc',
      type: boolean,
    },
    sortKey: {
      defaultValue: V1GetModelVersionsRequestSortBy.VERSION,
      storageKey: 'sortKey',
      type: union([
        literal(V1GetModelVersionsRequestSortBy.CREATIONTIME),
        literal(V1GetModelVersionsRequestSortBy.UNSPECIFIED),
        literal(V1GetModelVersionsRequestSortBy.VERSION),
      ]),
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
  storagePath: 'model-details',
};

export default config;
