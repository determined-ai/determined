import { isLeft } from 'fp-ts/lib/Either';
import * as io from 'io-ts';

import { ErrorLevel, ErrorType } from 'ErrorHandler';
import { CheckpointState, CheckpointStorageType, CommandState, LogLevel, RunState } from 'types';

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

const ioNullOrUndefined = io.union([ io.null, io.undefined ]);
const optional = (x: io.Mixed) => io.union([ x, ioNullOrUndefined ]);

/* User */

export const ioDetailedUser = io.type({
  active: io.boolean,
  admin: io.boolean,
  id: io.number,
  username: io.string,
});

export const ioDetailedUsers = io.array(ioDetailedUser);

export type ioTypeDetailedUsers = io.TypeOf<typeof ioDetailedUsers>;

export const ioLogin = io.type({ token: io.string });

export type ioTypeLogin = io.TypeOf<typeof ioLogin>;

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

const ioUser = io.type({
  id: io.number,
  username: io.string,
});

const ioCommandAddress = io.type({
  container_ip: io.string,
  container_port: io.number,
  host_ip: io.string,
  host_port: io.number,
  protocol: optional(io.string),
});

const ioCommandMisc = io.partial({
  experiment_ids: optional(io.array(io.number)),
  trial_ids: optional(io.array(io.number)),
});

const ioCommandConfig = io.exact(io.type({ description: io.string }));

const commandStates: Record<string, null> = Object.values(CommandState)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
const commandStatesIoType = io.keyof(commandStates);

export const ioGenericCommand = io.type({
  config: ioCommandConfig,
  exit_status: optional(io.string),
  id: io.string,
  misc: optional(ioCommandMisc),
  owner: ioUser,
  registered_time: io.string,
  service_address: optional(io.string),
  state: commandStatesIoType,
});

export const ioGenericCommands = io.record(io.string, ioGenericCommand);

export type ioTypeCommandAddress = io.TypeOf<typeof ioCommandAddress>;
export type ioTypeGenericCommand = io.TypeOf<typeof ioGenericCommand>;
export type ioTypeGenericCommands = io.TypeOf<typeof ioGenericCommands>;

const runStates: Record<string, null> = Object.values(RunState)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
const runStatesIoType = io.keyof(runStates);

/* Trials */

const checkpointStates: Record<string, null> = Object.values(CheckpointState)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
const checkpointStatesIoType = io.keyof(checkpointStates);

const ioMetricValue = io.any;
const ioMetric = io.record(io.string, ioMetricValue);
export type ioTypeMetric = io.TypeOf<typeof ioMetric>;

export const ioValidationMetrics = io.type({
  num_inputs: io.number,
  validation_metrics: ioMetric,
});
export type ioTypeValidationMetrics = io.TypeOf<typeof ioValidationMetrics>;

const startEndTimeDef = {
  end_time: optional(io.string),
  start_time: io.string,
};

export const ioCheckpoint = io.type({
  ...startEndTimeDef,
  id: io.number,
  resources: optional(io.record(io.string, io.number)),
  state: checkpointStatesIoType,
  step_id: io.number,
  trial_id: io.number,
  uuid: optional(io.string),
  validation_metric: ioMetricValue,
});
export type ioTypeCheckpoint = io.TypeOf<typeof ioCheckpoint>;

export const ioValidation = io.type({
  ...startEndTimeDef,
  id: io.number,
  metrics: optional(ioValidationMetrics),
  state: runStatesIoType,
});
export type ioTypeValidation = io.TypeOf<typeof ioValidation>;

export const ioStep = io.type({
  ...startEndTimeDef,
  avg_metrics: optional(ioMetric),
  checkpoint: optional(ioCheckpoint),
  id: io.number,
  num_batches: optional(io.number),
  prior_batches_processed: optional(io.number),
  state: runStatesIoType,
  validation: optional(ioValidation),
});
export type ioTypeStep = io.TypeOf<typeof ioStep>;

export const ioTrialDetails = io.type({
  end_time: optional(io.string),
  experiment_id: io.number,
  hparams: io.record(io.string, io.any),
  id: io.number,
  seed: io.number,
  start_time: io.string,
  state: runStatesIoType,
  steps: io.array(ioStep),
  warm_start_checkpoint_id: optional(io.number),
});
export type ioTypeTrialDetails = io.TypeOf<typeof ioTrialDetails>;

export const ioTrial = io.type({
  best_available_checkpoint: optional(ioCheckpoint),
  best_validation_metric: ioMetricValue,
  end_time: optional(io.string),
  experiment_id: io.number,
  hparams: io.record(io.string, io.any),
  id: io.number,
  latest_validation_metrics: optional(ioValidationMetrics),
  num_completed_checkpoints: io.number,
  num_steps: io.number,
  seed: io.number,
  start_time: io.string,
  state: runStatesIoType,
  total_batches_processed: optional(io.number),
});
export type ioTypeTrial = io.TypeOf<typeof ioTrial>;

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

const ioExpHParam = io.type({
  base: optional(io.number),
  count: optional(io.number),
  maxval: optional(io.number),
  minval: optional(io.number),
  type: io.keyof({ categorical: null, const: null, double: null, int: null, log: null }),
  val: optional(io.unknown),
});

export const ioHyperparameters = io.record(io.string, ioExpHParam);
export type ioTypeHyperparameters = io.TypeOf<typeof ioHyperparameters>;

export const ioExperimentConfig = io.type({
  checkpoint_policy: io.string,
  checkpoint_storage: optional(ioCheckpointStorage),
  data_layer: optional(ioDataLayer),
  description: io.string,
  hyperparameters: ioHyperparameters,
  labels: optional(io.array(io.string)),
  resources: ioExpResources,
  searcher: io.type({
    metric: io.string,
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
  start_time: io.string,
  state: runStatesIoType,
});

export const ioExperiments = io.array(ioExperiment);

export type ioTypeExperiment = io.TypeOf<typeof ioExperiment>;
export type ioTypeExperiments = io.TypeOf<typeof ioExperiments>;

const ioValidationHistory = io.type({
  end_time: io.string,
  trial_id: io.number,
  validation_error: optional(io.number),
});

export const ioExperimentDetails = io.type({
  archived: io.boolean,
  config: ioExperimentConfig,
  end_time: optional(io.string),
  id: io.number,
  owner: ioUser,
  progress: optional(io.number),
  start_time: io.string,
  state: runStatesIoType,
  trials: io.array(ioTrial),
  validation_history: io.array(ioValidationHistory),
});

export type ioTypeExperimentDetails = io.TypeOf<typeof ioExperimentDetails>;

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
