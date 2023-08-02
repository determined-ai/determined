import { array, boolean, literal, number, string, undefined as undefinedType, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';
import { V1GetExperimentCheckpointsRequestSortBy } from 'services/api-ts-sdk';
import { CheckpointState } from 'types';

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
  sortKey: V1GetExperimentCheckpointsRequestSortBy;
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
        union([
          literal('action'),
          literal('uuid'),
          literal('state'),
          literal('searcherMetric'),
          literal('totalBatches'),
          literal('checkpoint'),
        ]),
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
      defaultValue: V1GetExperimentCheckpointsRequestSortBy.UUID,
      storageKey: 'sortKey',
      type: union([
        literal(V1GetExperimentCheckpointsRequestSortBy.BATCHNUMBER),
        literal(V1GetExperimentCheckpointsRequestSortBy.ENDTIME),
        literal(V1GetExperimentCheckpointsRequestSortBy.SEARCHERMETRIC),
        literal(V1GetExperimentCheckpointsRequestSortBy.STATE),
        literal(V1GetExperimentCheckpointsRequestSortBy.TRIALID),
        literal(V1GetExperimentCheckpointsRequestSortBy.UNSPECIFIED),
        literal(V1GetExperimentCheckpointsRequestSortBy.UUID),
      ]),
    },
    state: {
      defaultValue: undefined,
      storageKey: 'state',
      type: union([
        undefinedType,
        array(
          union([
            literal(CheckpointState.Active),
            literal(CheckpointState.Completed),
            literal(CheckpointState.Deleted),
            literal(CheckpointState.PartiallyDeleted),
            literal(CheckpointState.Error),
            literal(CheckpointState.Unspecified),
          ]),
        ),
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
  storagePath: `${id}-checkpoints`,
});
