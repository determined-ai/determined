import { array, boolean, literal, number, string, undefined as undefinedType, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';
import { V1GetTemplatesRequestSortBy } from 'services/api-ts-sdk';

export type TemplateColumnName = 'action' | 'name' | 'workspace';

export const DEFAULT_COLUMNS: TemplateColumnName[] = ['name', 'workspace'];

export const DEFAULT_COLUMN_WIDTHS: Record<TemplateColumnName, number> = {
  action: 46,
  name: 150,
  workspace: 50,
};

export interface Settings extends InteractiveTableSettings {
  columns: TemplateColumnName[];
  workspace?: number[];
  name?: string;
  sortKey: string;
}

const config = (id: string): SettingsConfig<Settings> => ({
  settings: {
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(union([literal('name'), literal('workspace'), literal('action')])),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: TemplateColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
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
      defaultValue: V1GetTemplatesRequestSortBy.UNSPECIFIED,
      storageKey: 'sortKey',
      type: string,
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
    workspace: {
      defaultValue: undefined,
      storageKey: 'workspace',
      type: union([undefinedType, array(number)]),
    },
  },
  storagePath: `template-list-${id}`,
});

export default config;
