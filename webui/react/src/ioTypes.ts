import { isLeft } from 'fp-ts/lib/Either';
import * as io from 'io-ts';

import { ErrorLevel, ErrorType } from 'ErrorHandler';
import {
  CheckpointStorageType, ExperimentSearcherName, HyperparameterType, LogLevel, RunState,
} from 'types';

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const decode = <T>(type: io.Mixed, data: any): T => {
  try {
    const result = type.decode(data);
    if (isLeft(result)) throw result.left;
    return result.right;
  } catch (e) {
    const daError = {
      error: e,
      level: ErrorLevel.Fatal,
      silent: false,
      type: ErrorType.ApiBadResponse,
    };
    throw daError;
  }
};

const optional = (x: io.Mixed) => io.union([ x, io.null, io.undefined ]);

/* Info */

export const ioDeterminedInfo = io.type({
  cluster_id: io.string,
  cluster_name: io.string,
  isTelemetryEnabled: io.boolean,
  master_id: io.string,
  version: io.string,
});

export type ioTypeDeterminedInfo = io.TypeOf<typeof ioDeterminedInfo>;

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

const runStates: Record<string, null> = Object.values(RunState)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
const runStatesIoType = io.keyof(runStates);

/* Trials */

const ioMetricValue = io.any;
const ioMetric = io.record(io.string, ioMetricValue);
export type ioTypeMetric = io.TypeOf<typeof ioMetric>;

/* Experiments */

const checkpointStorageTypes: Record<string, null> = Object
  .values(CheckpointStorageType)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
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

const ioDataLayer = io.type({
  container_storage_path: optional(io.string),
  type: io.string,
});

const ioExpResources = io.type({ max_slots: optional(io.number) });

const hParamTypes: Record<string, null> = Object
  .values(HyperparameterType)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
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

const experimentSearcherName: Record<string, null> = Object.values(ExperimentSearcherName)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
export const ioExperimentConfig = io.type({
  checkpoint_policy: io.string,
  checkpoint_storage: optional(ioCheckpointStorage),
  data_layer: optional(ioDataLayer),
  description: optional(io.string),
  hyperparameters: ioHyperparameters,
  labels: optional(io.array(io.string)),
  name: io.string,
  profiling: optional(io.type({ enabled: io.boolean })),
  resources: ioExpResources,
  searcher: io.type({
    metric: io.string,
    name: io.keyof(experimentSearcherName),
    smaller_is_better: io.boolean,
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

const ioLogLevels: Record<string, null> = Object.values(LogLevel)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
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
  container_started_event: io.unknown,
  exited_event: optional(io.string),
  id: io.string,
  log_event: optional(io.string),
  parent_id: io.string,
  scheduled_event: optional(io.string),
  seq: io.number,
  service_ready_event: optional(io.type({})),
  snapshot: io.type({ config: io.type({ description: io.string }) }),
  terminate_request_event: optional(io.string),
  time: io.string,
});

export const ioTaskLogs = io.array(ioTaskLog);

export type ioTypeTaskLog = io.TypeOf<typeof ioTaskLog>;
export type ioTypeTaskLogs = io.TypeOf<typeof ioTaskLogs>;
