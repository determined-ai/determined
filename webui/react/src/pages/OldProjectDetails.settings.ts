import {
  array,
  boolean,
  literal,
  number,
  record,
  string,
  undefined as undefinedType,
  union,
} from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';
import { V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { RunState } from 'types';

export type ExperimentColumnName =
  | 'action'
  | 'archived'
  | 'description'
  | 'duration'
  | 'forkedFrom'
  | 'id'
  | 'name'
  | 'progress'
  | 'resourcePool'
  | 'searcherType'
  | 'startTime'
  | 'state'
  | 'tags'
  | 'numTrials'
  | 'user';

export const DEFAULT_COLUMNS: ExperimentColumnName[] = [
  'id',
  'name',
  'description',
  'tags',
  'forkedFrom',
  'startTime',
  'state',
  'searcherType',
  'user',
];

export const DEFAULT_COLUMN_WIDTHS: Record<ExperimentColumnName, number> = {
  action: 46,
  archived: 80,
  description: 148,
  duration: 96,
  forkedFrom: 128,
  id: 60,
  name: 150,
  numTrials: 74,
  progress: 111,
  resourcePool: 140,
  searcherType: 140,
  startTime: 118,
  state: 106,
  tags: 106,
  user: 85,
};

export interface ProjectDetailsSettings extends InteractiveTableSettings {
  archived?: boolean;
  columns: ExperimentColumnName[];
  label?: string[];
  pinned: Record<number, number[]>; // key is `projectId`, value is array of experimentId
  row?: number[];
  search?: string;
  sortKey: V1GetExperimentsRequestSortBy;
  state?: RunState[];
  type?: string[];
  user?: string[];
}

const config: SettingsConfig<ProjectDetailsSettings> = {
  applicableRoutespace: '/experiments',
  settings: {
    archived: {
      defaultValue: false,
      storageKey: 'archived',
      type: union([boolean, undefinedType]),
    },
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      storageKey: 'columns',
      type: array(
        union([
          literal('action'),
          literal('archived'),
          literal('description'),
          literal('duration'),
          literal('forkedFrom'),
          literal('id'),
          literal('name'),
          literal('progress'),
          literal('resourcePool'),
          literal('searcherType'),
          literal('startTime'),
          literal('state'),
          literal('tags'),
          literal('numTrials'),
          literal('user'),
        ]),
      ),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: ExperimentColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: array(number),
    },
    label: {
      defaultValue: undefined,
      storageKey: 'label',
      type: union([undefinedType, array(string)]),
    },
    pinned: {
      defaultValue: { 1: [] },
      skipUrlEncoding: true,
      storageKey: 'pinned',
      type: record(number, array(number)),
    },
    row: {
      defaultValue: undefined,
      storageKey: 'row',
      type: union([undefinedType, array(number)]),
    },
    search: {
      defaultValue: undefined,
      storageKey: 'search',
      type: union([undefinedType, string]),
    },
    sortDesc: {
      defaultValue: true,
      storageKey: 'sortDesc',
      type: boolean,
    },
    sortKey: {
      defaultValue: V1GetExperimentsRequestSortBy.STARTTIME,
      storageKey: 'sortKey',
      type: union([
        literal(V1GetExperimentsRequestSortBy.DESCRIPTION),
        literal(V1GetExperimentsRequestSortBy.ENDTIME),
        literal(V1GetExperimentsRequestSortBy.FORKEDFROM),
        literal(V1GetExperimentsRequestSortBy.ID),
        literal(V1GetExperimentsRequestSortBy.NAME),
        literal(V1GetExperimentsRequestSortBy.NUMTRIALS),
        literal(V1GetExperimentsRequestSortBy.PROGRESS),
        literal(V1GetExperimentsRequestSortBy.PROJECTID),
        literal(V1GetExperimentsRequestSortBy.RESOURCEPOOL),
        literal(V1GetExperimentsRequestSortBy.STARTTIME),
        literal(V1GetExperimentsRequestSortBy.STATE),
        literal(V1GetExperimentsRequestSortBy.UNSPECIFIED),
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
    type: {
      defaultValue: undefined,
      storageKey: 'type',
      type: union([undefinedType, array(string)]),
    },
    user: {
      defaultValue: undefined,
      storageKey: 'user',
      type: union([undefinedType, array(string)]),
    },
  },
  storagePath: 'project-details',
};

export default config;
