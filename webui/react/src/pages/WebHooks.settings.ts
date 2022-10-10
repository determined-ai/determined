import { InteractiveTableSettings } from 'components/InteractiveTable';
// import { MINIMUM_PAGE_SIZE } from 'components/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';

export type WebhookColumnName = 'id' | 'url';

export const DEFAULT_COLUMNS: WebhookColumnName[] = ['id', 'url'];

export const DEFAULT_COLUMN_WIDTHS: Record<WebhookColumnName, number> = {
  id: 46,
  url: 75,
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
  ],
  storagePath: 'webhook-list',
};

export default config;
