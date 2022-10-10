import { InteractiveTableSettings } from 'components/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';

export type WebhookColumnName = 'action' | 'test' | 'triggers' | 'url' | 'webhookType';

export const DEFAULT_COLUMNS: WebhookColumnName[] = ['url', 'webhookType', 'triggers', 'test'];

export const DEFAULT_COLUMN_WIDTHS: Record<WebhookColumnName, number> = {
  action: 30,
  test: 30,
  triggers: 60,
  webhookType: 60,
  url: 70,
};

export interface Settings extends InteractiveTableSettings {
  columns: WebhookColumnName[];
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
      defaultValue: DEFAULT_COLUMNS.map((col: WebhookColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      key: 'columnWidths',
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: {
        baseType: BaseType.Float,
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
  ],
  storagePath: 'webhook-list',
};

export default config;
