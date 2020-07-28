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
    segment_key: io.union([ io.string, io.undefined ]),
  }),
  version: io.string,
});

export type ioTypeDeterminedInfo = io.TypeOf<typeof ioDeterminedInfo>;

/* Slot */

export const ioSlotDevice = io.type({
  brand: io.string,
  id: io.number,
  type: io.string,
  uuid: io.union([ io.string, io.null ]),
});

export const ioSlotContainer = io.type({
  devices: io.array(ioSlotDevice),
  id: io.string,
  state: io.string,
});

export const ioSlot = io.type({
  container: io.union([ ioSlotContainer, io.null ]),
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
  protocol: io.union([ io.string, io.undefined ]),
});

const ioCommandMisc = io.partial({
  experiment_ids: io.union([ io.array(io.number), io.null ]),
  trial_ids: io.union([ io.array(io.number), io.null ]),
});

const ioCommandConfig = io.exact(io.type({
  description: io.string,
}));

const commandStates: Record<string, null> = Object.values(CommandState)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
const commandStatesIoType = io.keyof(commandStates);

export const ioGenericCommand = io.type({
  config: ioCommandConfig,
  exit_status: io.union([ io.string, io.null ]),
  id: io.string,
  misc: io.union([ ioCommandMisc, io.null ]),
  owner: ioOwner,
  registered_time: io.string,
  service_address: io.union([ io.string, io.null ]),
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

export const ioCheckpoint = io.type({
  end_time: io.union([ io.string, io.null ]),
  id: io.number,
  resources: io.record(io.string, io.number),
  start_time: io.string,
  state: checkpointStatesIoType,
  step_id: io.number,
  trial_id: io.number,
  uuid: io.union([ io.string, io.null ]),
  validation_metric: io.union([ io.number, io.undefined ]),
});
export type ioTypeCheckpoint = io.TypeOf<typeof ioCheckpoint>;

export const ioStep = io.type({
  end_time: io.union([ io.string, io.null ]),
  id: io.number,
  start_time: io.string,
  state: runStatesIoType,
});

export const ioTrialDetails = io.type({
  end_time: io.union([ io.string, io.null ]),
  experiment_id: io.number,

  id: io.number,

  seed: io.number,
  start_time: io.string,

  state: runStatesIoType,
  steps: io.array(ioStep),
  warm_start_checkpoint_id: io.union([ io.number, io.null ]),
});
export type ioTypeTrialDetails = io.TypeOf<typeof ioTrialDetails>;

export const ioLatestValidatonMetrics = io.type({
  num_inputs: io.number,
  validation_metrics: io.record(io.string, io.number),
});
export type ioTypeLatestValidationMetrics = io.TypeOf<typeof ioLatestValidatonMetrics>;

export const ioTrial = io.type({
  best_available_checkpoint: io.union([ ioCheckpoint, io.null ]),
  best_validation_metric: io.union([ io.number, io.null ]),
  end_time: io.union([ io.string, io.null ]),
  experiment_id: io.number,
  hparams: io.any,
  id: io.number,
  latest_validation_metrics: io.union([ ioLatestValidatonMetrics, io.null ]),
  num_batches: io.union([ io.number, io.null ]),
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
  bucket: io.union([ io.string, io.undefined ]),
  host_path: io.union([ io.string, io.undefined ]),
  save_experiment_best: io.number,
  save_trial_best: io.number,
  save_trial_latest: io.number,
  storage_path: io.union([ io.string, io.undefined ]),
  type: io.union([ ioCheckpointStorageType, io.undefined ]),
});

const ioDataLayer = io.type({
  container_storage_path: io.union([ io.string, io.null ]),
  type: io.string,
});

const ioExpResources = io.type({
  max_slots: io.union([ io.number, io.undefined ]),
});

export const ioExperimentConfig = io.type({
  checkpoint_policy: io.string,
  checkpoint_storage: io.union([ ioCheckpointStorage, io.null ]),
  data_layer: io.union([ ioDataLayer, io.undefined ]),
  description: io.string,
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
  end_time: io.union([ io.string, io.null ]),
  id: io.number,
  owner_id: io.number,
  progress: io.union([ io.number, io.null ]),
  start_time: io.string,
  state: runStatesIoType,
});

export const ioExperiments = io.array(ioExperiment);

export type ioTypeExperiment = io.TypeOf<typeof ioExperiment>;
export type ioTypeExperiments = io.TypeOf<typeof ioExperiments>;

const validationHistoryIoType = io.type({
  end_time: io.string,
  trial_id: io.number,
  validation_error: io.union([ io.number, io.null ]),
});

export const ioExperimentDetails = io.type({
  archived: io.boolean,
  config: ioExperimentConfig,
  end_time: io.union([ io.string, io.null ]),
  id: io.number,
  owner: ioOwner,
  progress: io.union([ io.number, io.null ]),
  start_time: io.string,
  state: runStatesIoType,
  trials: io.array(ioTrial),
  validation_history: io.array(validationHistoryIoType),
});

export type ioTypeExperimentDetails = io.TypeOf<typeof ioExperimentDetails>;

/* Logs */

const ioLogLevels: Record<string, null> = Object.values(LogLevel)
  .reduce((acc, val) => ({ ...acc, [val]: null }), {});
const ioLogLevelType = io.keyof(ioLogLevels);
export const ioLog = io.type({
  id: io.number,
  level: io.union([ ioLogLevelType, io.undefined ]),
  message: io.string,
  time: io.union([ io.string, io.undefined ]),
});

export const ioLogs = io.array(ioLog);

export type ioTypeLog = io.TypeOf<typeof ioLog>;
export type ioTypeLogs = io.TypeOf<typeof ioLogs>;

const ioCommandLogConfig = io.type({
  description: io.string,
});
const ioCommandLogSnapshot = io.type({
  config: ioCommandLogConfig,
});
const ioCommandLog = io.type({
  id: io.string,
  parent_id: io.string,
  seq: io.number,
  snapshot: ioCommandLogSnapshot,
  time: io.string,
});

export const ioCommandLogs = io.array(ioCommandLog);

export type ioTypeCommandLogs = io.TypeOf<typeof ioCommandLogs>;
