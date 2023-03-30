import { array, boolean, literal, number, undefined as undefinedType, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';
import { V1GetExperimentTrialsRequestSortBy } from 'services/api-ts-sdk';
import { RunState } from 'types';

export type TrialColumnName =
  | 'action'
  | 'id'
  | 'state'
  | 'totalBatchesProcessed'
  | 'bestValidationMetric'
  | 'latestValidationMetric'
  | 'startTime'
  | 'duration'
  | 'autoRestarts'
  | 'checkpoint'
  | 'totalCheckpointSize';

export const DEFAULT_COLUMNS: TrialColumnName[] = [
  'id',
  'state',
  'totalBatchesProcessed',
  'bestValidationMetric',
  'latestValidationMetric',
  'startTime',
  'duration',
  'autoRestarts',
  'checkpoint',
  'totalCheckpointSize',
];

export const DEFAULT_COLUMN_WIDTHS: Record<TrialColumnName, number> = {
  action: 46,
  autoRestarts: 124,
  bestValidationMetric: 190,
  checkpoint: 100,
  duration: 117,
  id: 85,
  latestValidationMetric: 200,
  startTime: 117,
  state: 100,
  totalBatchesProcessed: 90,
  totalCheckpointSize: 160,
};

export const isOfSortKey = (sortKey: React.Key): sortKey is V1GetExperimentTrialsRequestSortBy => {
  return Object.values<string>(V1GetExperimentTrialsRequestSortBy).includes(String(sortKey));
};

export interface Settings extends InteractiveTableSettings {
  columns: TrialColumnName[];
  compare: boolean;
  row?: number[];
  sortDesc: boolean;
  sortKey: V1GetExperimentTrialsRequestSortBy;
  state?: RunState[];
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
          literal('id'),
          literal('state'),
          literal('totalBatchesProcessed'),
          literal('bestValidationMetric'),
          literal('latestValidationMetric'),
          literal('startTime'),
          literal('duration'),
          literal('autoRestarts'),
          literal('checkpoint'),
          literal('totalCheckpointSize'),
        ]),
      ),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: TrialColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: array(number),
    },
    compare: {
      defaultValue: false,
      storageKey: 'compare',
      type: boolean,
    },
    row: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'row',
      type: union([undefinedType, array(number)]),
    },
    sortDesc: {
      defaultValue: true,
      storageKey: 'sortDesc',
      type: boolean,
    },
    sortKey: {
      defaultValue: V1GetExperimentTrialsRequestSortBy.ID,
      storageKey: 'sortKey',
      type: union([
        literal(V1GetExperimentTrialsRequestSortBy.BATCHESPROCESSED),
        literal(V1GetExperimentTrialsRequestSortBy.BESTVALIDATIONMETRIC),
        literal(V1GetExperimentTrialsRequestSortBy.DURATION),
        literal(V1GetExperimentTrialsRequestSortBy.ENDTIME),
        literal(V1GetExperimentTrialsRequestSortBy.ID),
        literal(V1GetExperimentTrialsRequestSortBy.LATESTVALIDATIONMETRIC),
        literal(V1GetExperimentTrialsRequestSortBy.RESTARTS),
        literal(V1GetExperimentTrialsRequestSortBy.STARTTIME),
        literal(V1GetExperimentTrialsRequestSortBy.STATE),
        literal(V1GetExperimentTrialsRequestSortBy.UNSPECIFIED),
        literal(V1GetExperimentTrialsRequestSortBy.CHECKPOINTSIZE),
      ]),
    },
    state: {
      defaultValue: undefined,
      storageKey: 'state',
      type: union([
        undefinedType,
        array(
          union([
            literal(RunState.Active),
            literal(RunState.Canceled),
            literal(RunState.Completed),
            literal(RunState.DeleteFailed),
            literal(RunState.Deleted),
            literal(RunState.Deleting),
            literal(RunState.Error),
            literal(RunState.Paused),
            literal(RunState.Pulling),
            literal(RunState.Queued),
            literal(RunState.Running),
            literal(RunState.Starting),
            literal(RunState.StoppingCanceled),
            literal(RunState.StoppingCompleted),
            literal(RunState.StoppingError),
            literal(RunState.StoppingKilled),
            literal(RunState.Unspecified),
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
  storagePath: `${id}-experiment-trials-list`,
});
