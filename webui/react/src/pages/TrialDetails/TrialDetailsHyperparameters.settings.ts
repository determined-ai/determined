import { array, boolean, literal, number, string, undefined as undefinedType, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { SettingsConfig } from 'hooks/useSettings';

export type HyperparameterColumnName = 'hyperparameter' | 'value';

export const DEFAULT_COLUMNS: HyperparameterColumnName[] = ['hyperparameter', 'value'];

export const DEFAULT_COLUMN_WIDTHS: Record<HyperparameterColumnName, number> = {
  hyperparameter: 150,
  value: 150,
};

export interface Settings extends InteractiveTableSettings {
  columns: HyperparameterColumnName[];
}

export const configForTrial = (id: number): SettingsConfig<Settings> => ({
  settings: {
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(union([literal('hyperparameter'), literal('value')])),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map(
        (col: HyperparameterColumnName) => DEFAULT_COLUMN_WIDTHS[col],
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
      defaultValue: 'hyperparameter',
      skipUrlEncoding: true,
      storageKey: 'sortKey',
      type: union([undefinedType, union([boolean, number, string])]),
    },
    tableLimit: {
      defaultValue: 0,
      storageKey: 'tableLimit',
      type: number,
    },
    tableOffset: {
      defaultValue: 0,
      storageKey: 'tableOffset',
      type: number,
    },
  },
  storagePath: `trial-${id}-hyperparameters`,
});
