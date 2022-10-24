import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { BaseType, SettingsConfig } from 'hooks/useSettings';

export type HyperparameterColumnName = 'hyperparameter' | 'value';

export const DEFAULT_COLUMNS: HyperparameterColumnName[] = ['hyperparameter', 'value'];

export const DEFAULT_COLUMN_WIDTHS: Record<HyperparameterColumnName, number> = {
  hyperparameter: 150,
  value: 150,
};

export interface Settings extends InteractiveTableSettings {
  columns: HyperparameterColumnName[];
}

const config: SettingsConfig = {
  applicableRoutespace: '/hyperparameters',
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
        (col: HyperparameterColumnName) => DEFAULT_COLUMN_WIDTHS[col],
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
      defaultValue: 'hyperparameter',
      key: 'sortKey',
      skipUrlEncoding: true,
      storageKey: 'sortKey',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'trial-hyperparameters',
};

export default config;
