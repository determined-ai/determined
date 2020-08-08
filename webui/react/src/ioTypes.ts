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

/* User */

export const ioUser = io.type({
  active: io.boolean,
  admin: io.boolean,
  id: io.number,
  username: io.string,
});

export const ioUsers = io.array(ioUser);

export type ioTypeUser = io.TypeOf<typeof ioUser>;
export type ioTypeUsers = io.TypeOf<typeof ioUsers>;

/* Info */

export const ioDeterminedInfo = io.type({
  cluster_id: io.string,
  master_id: io.string,
  telemetry: io.type({
    enabled: io.boolean,
    segment_key: io.union([ io.string, ioNullOrUndefined ]),
  }),
  version: io.string,
});

export type ioTypeDeterminedInfo = io.TypeOf<typeof ioDeterminedInfo>;

/* Slot */

export const ioSlotDevice = io.type({
  brand: io.string,
  id: io.number,
  type: io.string,
  uuid: io.union([ io.string, ioNullOrUndefined ]),
});

export const ioSlotContainer = io.type({
  devices: io.union([ io.array(ioSlotDevice), ioNullOrUndefined ]),
  id: io.string,
  state: io.string,
});

export const ioSlot = io.type({
  container: io.union([ ioSlotContainer, ioNullOrUndefined ]),
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

const ioOwner = io.type({
  id: io.number,
  username: io.string,
});

const ioCommandAddress = io.type({
  container_ip: io.string,
  container_port: io.number,
  host_ip: io.string,
  host_port: io.number,
  protocol: io.union([ io.string, ioNullOrUndefined ]),
});

const ioCommandMisc = io.partial({
  experiment_ids: io.union([ io.array(io.number), ioNullOrUndefined ]),
  trial_ids: io.union([ io.array(io.number), ioNullOrUndefined ]),
});

const ioCommandConfig = io.exact(io.type({
  description: io.string,
}));

const commandStates: Record<string, null> = Object.values(CommandState)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
const commandStatesIoType = io.keyof(commandStates);

export const ioGenericCommand = io.type({
  config: ioCommandConfig,
  exit_status: io.union([ io.string, ioNullOrUndefined ]),
  id: io.string,
  misc: io.union([ ioCommandMisc, ioNullOrUndefined ]),
  owner: ioOwner,
  registered_time: io.string,
  service_address: io.union([ io.string, ioNullOrUndefined ]),
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

export const ioValidationMetrics = io.type({
  num_inputs: io.number,
  validation_metrics: io.record(io.string, io.number),
});
export type ioTypeValidationMetrics = io.TypeOf<typeof ioValidationMetrics>;

const startEndTimeDef = {
  end_time: io.union([ io.string, ioNullOrUndefined ]),
  start_time: io.string,
};

const baseStepDef = {
  ...startEndTimeDef,
  id: io.number,
  trial_id: io.number,
};

export const ioCheckpoint = io.type({
  ...baseStepDef,
  resources: io.record(io.string, io.number),
  state: checkpointStatesIoType,
  step_id: io.number,
  trial_id: io.number,
  uuid: io.union([ io.string, ioNullOrUndefined ]),
  validation_metric: io.union([ io.number, ioNullOrUndefined ]),
});
export type ioTypeCheckpoint = io.TypeOf<typeof ioCheckpoint>;

export const ioValidation = io.type({
  ...baseStepDef,
  metrics: io.union([ io.null, ioValidationMetrics ]),
  state: runStatesIoType,
  step_id: io.number,
});
export type ioTypeValidation = io.TypeOf<typeof ioValidation>;

export const ioStep = io.type({
  ...baseStepDef,
  checkpoint: io.union([ ioCheckpoint, ioNullOrUndefined ]),
  state: runStatesIoType,
  validation: io.union([ ioValidation, ioNullOrUndefined ]),
});
export type ioTypeStep = io.TypeOf<typeof ioStep>;

export const ioTrialDetails = io.type({
  end_time: io.union([ io.string, ioNullOrUndefined ]),
  experiment_id: io.number,
  hparams: io.record(io.string, io.any),
  id: io.number,
  seed: io.number,
  start_time: io.string,
  state: runStatesIoType,
  steps: io.array(ioStep),
  warm_start_checkpoint_id: io.union([ io.number, ioNullOrUndefined ]),
});
export type ioTypeTrialDetails = io.TypeOf<typeof ioTrialDetails>;

export const ioTrial = io.type({
  best_available_checkpoint: io.union([ ioCheckpoint, ioNullOrUndefined ]),
  best_validation_metric: io.union([ io.number, ioNullOrUndefined ]),
  end_time: io.union([ io.string, ioNullOrUndefined ]),
  experiment_id: io.number,
  hparams: io.record(io.string, io.any),
  id: io.number,
  latest_validation_metrics: io.union([ ioValidationMetrics, ioNullOrUndefined ]),
  num_batches: io.union([ io.number, ioNullOrUndefined ]),
  num_completed_checkpoints: io.number,
  num_steps: io.number,
  seed: io.number,
  start_time: io.string,
  state: runStatesIoType,
});
export type ioTypeTrial = io.TypeOf<typeof ioTrial>;

/* Experiments */

const checkpointStorageTypes: Record<string, null> = Object
  .values(CheckpointStorageType)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
const ioCheckpointStorageType = io.keyof(checkpointStorageTypes);

export const ioCheckpointStorage = io.type({
  bucket: io.union([ io.string, ioNullOrUndefined ]),
  host_path: io.union([ io.string, ioNullOrUndefined ]),
  save_experiment_best: io.number,
  save_trial_best: io.number,
  save_trial_latest: io.number,
  storage_path: io.union([ io.string, ioNullOrUndefined ]),
  type: io.union([ ioCheckpointStorageType, ioNullOrUndefined ]),
});

const ioDataLayer = io.type({
  container_storage_path: io.union([ io.string, ioNullOrUndefined ]),
  type: io.string,
});

const ioExpResources = io.type({
  max_slots: io.union([ io.number, ioNullOrUndefined ]),
});

const ioExperimentConfig = io.type({
  checkpoint_policy: io.string,
  checkpoint_storage: io.union([ ioCheckpointStorage, ioNullOrUndefined ]),
  data_layer: io.union([ ioDataLayer, ioNullOrUndefined ]),
  description: io.string,
  labels: io.union([ io.array(io.string), ioNullOrUndefined ]),
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
  end_time: io.union([ io.string, ioNullOrUndefined ]),
  id: io.number,
  owner_id: io.number,
  progress: io.union([ io.number, ioNullOrUndefined ]),
  start_time: io.string,
  state: runStatesIoType,
});

export const ioExperiments = io.array(ioExperiment);

export type ioTypeExperiment = io.TypeOf<typeof ioExperiment>;
export type ioTypeExperiments = io.TypeOf<typeof ioExperiments>;

const ioValidationHistory = io.type({
  end_time: io.string,
  trial_id: io.number,
  validation_error: io.union([ io.number, ioNullOrUndefined ]),
});

export const ioExperimentDetails = io.type({
  archived: io.boolean,
  config: ioExperimentConfig,
  end_time: io.union([ io.string, ioNullOrUndefined ]),
  id: io.number,
  owner: ioOwner,
  progress: io.union([ io.number, ioNullOrUndefined ]),
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
  level: io.union([ ioLogLevelType, ioNullOrUndefined ]),
  message: io.string,
  time: io.union([ io.string, ioNullOrUndefined ]),
});

export const ioLogs = io.array(ioLog);

export type ioTypeLog = io.TypeOf<typeof ioLog>;
export type ioTypeLogs = io.TypeOf<typeof ioLogs>;

const ioTaskLog = io.type({
  assigned_event: io.union([
    io.type({ NumContainers: io.number }),
    ioNullOrUndefined,
  ]),
  container_started_event: io.union([
    io.type({ Container: io.type({}) }),
    ioNullOrUndefined,
  ]),
  exited_event: io.union([ io.string, ioNullOrUndefined ]),
  id: io.string,
  log_event: io.union([ io.string, ioNullOrUndefined ]),
  parent_id: io.string,
  scheduled_event: io.union([ io.string, ioNullOrUndefined ]),
  seq: io.number,
  service_ready_event: io.union([ io.type({}), ioNullOrUndefined ]),
  snapshot: io.type({
    config: io.type({
      description: io.string,
    }),
  }),
  terminate_request_event: io.union([ io.string, ioNullOrUndefined ]),
  time: io.string,
});

export const ioTaskLogs = io.array(ioTaskLog);

export type ioTypeTaskLog = io.TypeOf<typeof ioTaskLog>;
export type ioTypeTaskLogs = io.TypeOf<typeof ioTaskLogs>;
