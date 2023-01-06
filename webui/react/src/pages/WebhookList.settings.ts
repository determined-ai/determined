import { array, boolean, literal, number, string, undefined as undefinedType, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';

export type WebhookColumnName = 'action' | 'triggers' | 'url' | 'webhookType';

export const DEFAULT_COLUMNS: WebhookColumnName[] = ['url', 'webhookType', 'triggers'];

export const DEFAULT_COLUMN_WIDTHS: Record<WebhookColumnName, number> = {
  action: 30,
  triggers: 150,
  url: 150,
  webhookType: 60,
};

export interface Settings extends InteractiveTableSettings {
  columns: WebhookColumnName[];
}

const config: SettingsConfig<Settings> = {
  settings: {
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(
        union([literal('action'), literal('triggers'), literal('url'), literal('webhookType')]),
      ),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: WebhookColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
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
      defaultValue: 'id',
      storageKey: 'sortKey',
      type: union([undefinedType, number, string, boolean]),
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
  storageKey: 'webhook-list',
};

export default config;
