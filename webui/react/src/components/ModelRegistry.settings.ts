import { array, boolean, literal, number, string, undefined as undefinedType, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';

export type ModelColumnName =
  | 'action'
  | 'archived'
  | 'description'
  | 'lastUpdatedTime'
  | 'name'
  | 'tags'
  | 'numVersions'
  | 'user'
  | 'workspace';

export const DEFAULT_COLUMNS: ModelColumnName[] = [
  'name',
  'description',
  'workspace',
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
  workspace: 130,
};
export const isOfSortKey = (sortKey: React.Key): sortKey is V1GetModelsRequestSortBy => {
  const sortKeys = [...Object.values(V1GetModelsRequestSortBy), 'workspace'];
  return sortKeys.includes(String(sortKey));
};

export interface Settings extends InteractiveTableSettings {
  archived?: boolean;
  columns: ModelColumnName[];
  description?: string;
  name?: string;
  sortKey: V1GetModelsRequestSortBy;
  tags?: string[];
  users?: string[];
  workspace?: number[];
}

const config = (id: string): SettingsConfig<Settings> => {
  const storagePath = `model-registry-${id}`;

  return {
    settings: {
      archived: {
        defaultValue: undefined,
        storageKey: 'archived',
        type: union([undefinedType, boolean]),
      },
      columns: {
        defaultValue: DEFAULT_COLUMNS,
        skipUrlEncoding: true,
        storageKey: 'columns',
        type: array(
          union([
            literal('action'),
            literal('archived'),
            literal('description'),
            literal('lastUpdatedTime'),
            literal('name'),
            literal('tags'),
            literal('numVersions'),
            literal('user'),
            literal('workspace'),
          ]),
        ),
      },
      columnWidths: {
        defaultValue: DEFAULT_COLUMNS.map((col: ModelColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
        skipUrlEncoding: true,
        storageKey: 'columnWidths',
        type: array(number),
      },
      description: {
        defaultValue: undefined,
        storageKey: 'description',
        type: union([undefinedType, string]),
      },
      name: {
        defaultValue: undefined,
        storageKey: 'name',
        type: union([undefinedType, string]),
      },
      sortDesc: {
        defaultValue: true,
        storageKey: 'sortDesc',
        type: boolean,
      },
      sortKey: {
        defaultValue: V1GetModelsRequestSortBy.CREATIONTIME,
        storageKey: 'sortKey',
        type: union([
          literal(V1GetModelsRequestSortBy.CREATIONTIME),
          literal(V1GetModelsRequestSortBy.UNSPECIFIED),
          literal(V1GetModelsRequestSortBy.LASTUPDATEDTIME),
          literal(V1GetModelsRequestSortBy.NAME),
          literal(V1GetModelsRequestSortBy.NUMVERSIONS),
          literal(V1GetModelsRequestSortBy.UNSPECIFIED),
          literal(V1GetModelsRequestSortBy.DESCRIPTION),
          literal(V1GetModelsRequestSortBy.WORKSPACE),
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
      tags: {
        defaultValue: undefined,
        storageKey: 'tags',
        type: union([undefinedType, array(string)]),
      },
      users: {
        defaultValue: undefined,
        storageKey: 'users',
        type: union([undefinedType, array(string)]),
      },
      workspace: {
        defaultValue: [],
        storageKey: 'workspace',
        type: union([undefinedType, array(number)]),
      },
    },
    storagePath,
  };
};

export default config;
