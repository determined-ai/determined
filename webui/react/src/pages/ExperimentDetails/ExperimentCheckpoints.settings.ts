import { array, boolean, keyof, number, string, undefined as undefinedType, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';
import { Checkpointv1SortBy } from 'services/api-ts-sdk';
import { CheckpointState } from 'types';
import valueof from 'utils/valueof';

export type CheckpointColumnName =
  | 'action'
  | 'uuid'
  | 'state'
  | 'searcherMetric'
  | 'totalBatches'
  | 'checkpoint';

export const DEFAULT_COLUMNS: CheckpointColumnName[] = [
  'uuid',
  'state',
  'totalBatches',
  'searcherMetric',
  'checkpoint',
];

export const DEFAULT_COLUMN_WIDTHS: Record<CheckpointColumnName, number> = {
  action: 46,
  checkpoint: 100,
  searcherMetric: 100,
  state: 117,
  totalBatches: 74,
  uuid: 200,
};

export interface Settings extends InteractiveTableSettings {
  columns: CheckpointColumnName[];
  row?: string[];
  sortDesc: boolean;
  sortKey: Checkpointv1SortBy;
  state?: CheckpointState[];
  tableLimit: number;
  tableOffset: number;
}

export const configForExperiment = (id: number): SettingsConfig<Settings> => ({
  settings: {
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(
        keyof({
          action: null,
          checkpoint: null,
          searcherMetric: null,
          state: null,
          totalBatches: null,
          uuid: null,
        }),
      ),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: CheckpointColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: array(number),
    },
    row: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'row',
      type: union([undefinedType, array(string)]),
    },
    sortDesc: {
      defaultValue: true,
      storageKey: 'sortDesc',
      type: boolean,
    },
    sortKey: {
      defaultValue: Checkpointv1SortBy.UUID,
      storageKey: 'sortKey',
      type: valueof(Checkpointv1SortBy),
    },
    state: {
      defaultValue: undefined,
      storageKey: 'state',
      type: union([undefinedType, array(valueof(CheckpointState))]),
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
  storagePath: `${id}-checkpoints`,
});

export const configForTrial = (id: number): SettingsConfig<Settings> => ({
  settings: {
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(
        keyof({
          action: null,
          checkpoint: null,
          searcherMetric: null,
          state: null,
          totalBatches: null,
          uuid: null,
        }),
      ),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: CheckpointColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: array(number),
    },
    row: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'row',
      type: union([undefinedType, array(string)]),
    },
    sortDesc: {
      defaultValue: true,
      storageKey: 'sortDesc',
      type: boolean,
    },
    sortKey: {
      defaultValue: Checkpointv1SortBy.UUID,
      storageKey: 'sortKey',
      type: valueof(Checkpointv1SortBy),
    },
    state: {
      defaultValue: undefined,
      storageKey: 'state',
      type: union([undefinedType, array(valueof(CheckpointState))]),
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
  storagePath: `trial-${id}-checkpoints`,
});
