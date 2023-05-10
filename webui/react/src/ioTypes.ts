import { isLeft } from 'fp-ts/lib/Either';
import * as io from 'io-ts';

import { V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import {
  CheckpointStorageType,
  ExperimentSearcherName,
  HyperparameterType,
  LogLevel,
  RunState,
} from 'types';

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const decode = <T>(type: io.Mixed, data: any): T => {
  try {
    const result = type.decode(data);
    if (isLeft(result)) throw result.left;
    return result.right;
  } catch (e) {
    throw new DetError(e, {
      level: ErrorLevel.Fatal,
      silent: false,
      type: ErrorType.ApiBadResponse,
    });
  }
};

export const optional = (x: io.Mixed): io.Mixed | io.NullC | io.UndefinedC => {
  return io.union([x, io.null, io.undefined]);
};

/* Slot */

export const ioSlotDevice = io.type({
  brand: io.string,
  id: io.number,
  type: io.string,
  uuid: optional(io.string),
});

export const ioSlotContainer = io.type({
  devices: optional(io.array(ioSlotDevice)),
  id: io.string,
  state: io.string,
});

export const ioSlot = io.type({
  container: optional(ioSlotContainer),
  device: ioSlotDevice,
  enabled: io.boolean,
  id: io.string,
});

export const ioSlots = io.record(io.string, ioSlot);

/* Agent */

export const ioAgent = io.type({
  id: io.string,
  registered_time: io.string,
  slots: ioSlots,
});

export const ioAgents = io.record(io.string, ioAgent);

export type ioTypeAgent = io.TypeOf<typeof ioAgent>;
export type ioTypeAgents = io.TypeOf<typeof ioAgents>;

/* Generic Command */

const runStates: Record<string, null> = Object.values(RunState).reduce(
  (acc, val) => ({ ...acc, [val]: null }),
  {},
);
const runStatesIoType = io.keyof(runStates);

/* Trials */

const ioMetricValue = io.unknown;
const ioMetric = io.record(io.string, ioMetricValue);
export type ioTypeMetric = io.TypeOf<typeof ioMetric>;

/* Experiments */

const checkpointStorageTypes: Record<string, null> = Object.values(CheckpointStorageType).reduce(
  (acc, val) => ({ ...acc, [val]: null }),
  {},
);
const ioCheckpointStorageType = io.keyof(checkpointStorageTypes);

export const ioCheckpointStorage = io.type({
  bucket: optional(io.string),
  host_path: optional(io.string),
  save_experiment_best: io.number,
  save_trial_best: io.number,
  save_trial_latest: io.number,
  storage_path: optional(io.string),
  type: optional(ioCheckpointStorageType),
});

const ioExpResources = io.type({ max_slots: optional(io.number) });

const hParamTypes: Record<string, null> = Object.values(HyperparameterType).reduce(
  (acc, val) => ({ ...acc, [val]: null }),
  {},
);
const ioHParamTypes = io.keyof(hParamTypes);
const ioExpHParamVal = optional(io.unknown);
const ioExpHParam = io.type({
  base: optional(io.number),
  count: optional(io.number),
  maxval: optional(io.number),
  minval: optional(io.number),
  type: ioHParamTypes,
  val: ioExpHParamVal,
  vals: optional(io.array(io.unknown)),
});

export type ioTypeHyperparameter = io.TypeOf<typeof ioExpHParam>;

/*
 * We are unable to create a recursive dictionary type in io-ts,
 * so until we have JavaScript JSON schema support:
 *   - temporarily changing to an unknown record
 *   - use a custom decoder instead of relying on io-ts to decode hp
 */
export const ioHyperparameters = io.UnknownRecord;
export type ioTypeHyperparameters = io.TypeOf<typeof ioHyperparameters>;

const experimentSearcherName: Record<string, null> = Object.values(ExperimentSearcherName).reduce(
  (acc, val) => ({ ...acc, [val]: null }),
  {},
);
export const ioExperimentConfig = io.type({
  checkpoint_policy: io.string,
  checkpoint_storage: optional(ioCheckpointStorage),
  description: optional(io.string),
  hyperparameters: ioHyperparameters,
  labels: optional(io.array(io.string)),
  max_restarts: io.number,
  name: io.string,
  profiling: optional(io.type({ enabled: io.boolean })),
  resources: ioExpResources,
  searcher: io.type({
    metric: io.string,
    name: io.keyof(experimentSearcherName),
    smaller_is_better: io.boolean,
    source_trial_id: io.union([io.null, io.number]),
  }),
});
export type ioTypeExperimentConfig = io.TypeOf<typeof ioExperimentConfig>;

export const ioExperiment = io.type({
  archived: io.boolean,
  config: ioExperimentConfig,
  end_time: optional(io.string),
  id: io.number,
  owner_id: io.number,
  progress: optional(io.number),
  resource_pool: io.string,
  start_time: io.string,
  state: runStatesIoType,
});

export const ioExperiments = io.array(ioExperiment);

export type ioTypeExperiment = io.TypeOf<typeof ioExperiment>;
export type ioTypeExperiments = io.TypeOf<typeof ioExperiments>;

/* Logs */

const ioLogLevels: Record<string, null> = Object.values(LogLevel).reduce(
  (acc, val) => ({ ...acc, [val]: null }),
  {},
);
const ioLogLevelType = io.keyof(ioLogLevels);
export const ioLog = io.type({
  id: io.number,
  level: optional(ioLogLevelType),
  message: io.string,
  time: optional(io.string),
});

export const ioLogs = io.array(ioLog);

export type ioTypeLog = io.TypeOf<typeof ioLog>;
export type ioTypeLogs = io.TypeOf<typeof ioLogs>;

const ioTaskLog = io.type({
  assigned_event: io.unknown,
  description: io.string,
  exited_event: optional(io.string),
  id: io.string,
  log_event: optional(io.string),
  parent_id: io.string,
  resources_started_event: io.unknown,
  scheduled_event: optional(io.string),
  seq: io.number,
  service_ready_event: optional(io.type({})),
  terminate_request_event: optional(io.string),
  time: io.string,
});

export const ioTaskLogs = io.array(ioTaskLog);

export type ioTypeTaskLog = io.TypeOf<typeof ioTaskLog>;
export type ioTypeTaskLogs = io.TypeOf<typeof ioTaskLogs>;

const ioLocationType = io.keyof({
  [V1LocationType.EXPERIMENT]: null,
  [V1LocationType.HYPERPARAMETERS]: null,
  [V1LocationType.VALIDATIONS]: null,
  [V1LocationType.UNSPECIFIED]: null,
});
const ioColumnType = io.keyof({
  [V1ColumnType.DATE]: null,
  [V1ColumnType.NUMBER]: null,
  [V1ColumnType.TEXT]: null,
  [V1ColumnType.UNSPECIFIED]: null,
});
const ioProjectColumnRequired = io.type({
  column: io.string,
  location: ioLocationType,
  type: ioColumnType,
});
const ioProjectColumnOptionals = io.partial({
  displayName: io.string,
});
const ioProjectColumn = io.intersection([ioProjectColumnRequired, ioProjectColumnOptionals]);
const ioProjectColumns = io.array(ioProjectColumn);
export const ioProjectColumnsResponse = io.type({
  columns: ioProjectColumns,
});

export type ioTypeProjectColumnsResponse = io.TypeOf<typeof ioProjectColumnsResponse>;
