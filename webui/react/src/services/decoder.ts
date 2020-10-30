import { isNumber } from 'util';

import dayjs from 'dayjs';

import {
  decode, ioAgents, ioDetailedUsers, ioDeterminedInfo,
  ioExperiment, ioExperimentDetails, ioExperiments, ioGenericCommand, ioGenericCommands, ioLog,
  ioLogs, ioTaskLogs, ioTrialDetails, ioTypeAgents,
  ioTypeCheckpoint, ioTypeDetailedUsers, ioTypeDeterminedInfo,
  ioTypeExperiment, ioTypeExperimentConfig, ioTypeExperimentDetails, ioTypeExperiments,
  ioTypeGenericCommand, ioTypeGenericCommands, ioTypeLog, ioTypeLogs, ioTypeMetric, ioTypeStep,
  ioTypeTaskLogs, ioTypeTrial, ioTypeTrialDetails, ioTypeValidationMetrics,
} from 'ioTypes';
import { waitPageUrl } from 'routes/utils';
import {
  Agent, Checkpoint, CheckpointState, CheckpointStorageType, Command, CommandState,
  CommandType, DetailedUser, DeterminedInfo, ExperimentBase,
  ExperimentConfig, ExperimentDetails, ExperimentItem, Log, LogLevel, RawJson,
  ResourceState, ResourceType, RunState, Step, TrialDetails, TrialItem, ValidationMetrics,
} from 'types';
import { capitalize } from 'utils/string';

import * as AB from './api-ts-sdk'; // API Bindings

const dropNonNumericMetrics = (ioMetrics: ioTypeMetric): Record<string, number> => {
  const metrics: Record<string, number> = {};
  Object.entries(ioMetrics).forEach(([ name, value ]) => {
    if (isNumber(value)) metrics[name] = value;
  });
  return metrics;
};

export const user = (data: AB.V1User): DetailedUser => {
  return {
    isActive: data.active,
    isAdmin: data.admin,
    username: data.username,
  };
};

export const jsonToUsers = (data: unknown): DetailedUser[] => {
  const io = decode<ioTypeDetailedUsers>(ioDetailedUsers, data);
  return io.map(user => ({
    id: user.id,
    isActive: user.active,
    isAdmin: user.admin,
    username: user.username,
  }));
};

export const jsonToDeterminedInfo = (data: unknown): DeterminedInfo => {
  const io = decode<ioTypeDeterminedInfo>(ioDeterminedInfo, data);
  return {
    clusterId: io.cluster_id,
    clusterName: io.cluster_name,
    masterId: io.master_id,
    telemetry: {
      enabled: io.telemetry.enabled,
      segmentKey: io.telemetry.segment_key || undefined,
    },
    version: io.version,
  };
};

export const jsonToAgents = (data: unknown): Agent[] => {
  const io = decode<ioTypeAgents>(ioAgents, data);
  return Object.keys(io).map(agentId => {
    const agent = io[agentId];
    const resources = Object.keys(agent.slots).map(slotId => {
      const slot = agent.slots[slotId];

      return {
        container: slot.container ? {
          id: slot.container.id,
          state: ResourceState[
            capitalize(slot.container.state) as keyof typeof ResourceState
          ],
        } : undefined,
        enabled: slot.enabled,
        id: slot.id,
        name: slot.device.brand,
        type: ResourceType[slot.device.type.toUpperCase() as keyof typeof ResourceType],
        uuid: slot.device.uuid || undefined,
      };
    });

    return {
      id: agent.id,
      registeredTime: dayjs(agent.registered_time).unix(),
      resources,
    };
  });
};

export const jsonToGenericCommand = (data: unknown, type: CommandType): Command => {
  const io = decode<ioTypeGenericCommand>(ioGenericCommand, data);
  const command: Partial<Command> = {
    config: { ...io.config },
    exitStatus: io.exit_status || undefined,
    id: io.id,
    kind: type,
    misc: io.misc ? {
      experimentIds: io.misc.experiment_ids || [],
      trialIds: io.misc.trial_ids || [],
    } : undefined,
    registeredTime: io.registered_time,
    serviceAddress: io.service_address || undefined,
    state: io.state as CommandState,
    url: undefined,
    user: { username: io.owner.username },
  };
  command.url = waitPageUrl(command);
  return command as Command;
};

const jsonToGenericCommands = (data: unknown, type: CommandType): Command[] => {
  const io = decode<ioTypeGenericCommands>(ioGenericCommands, data);
  return Object.keys(io).map(genericCommandId => {
    return jsonToGenericCommand(io[genericCommandId], type);
  });
};

export const jsonToCommands = (data: unknown): Command[] => {
  return jsonToGenericCommands(data, CommandType.Command);
};

export const jsonToNotebook = (data: unknown): Command => {
  return jsonToGenericCommand(data, CommandType.Notebook);
};

export const jsonToNotebooks = (data: unknown): Command[] => {
  return jsonToGenericCommands(data, CommandType.Notebook);
};

export const jsonToShells = (data: unknown): Command[] => {
  return jsonToGenericCommands(data, CommandType.Shell);
};

export const jsonToTensorboard = (data: unknown): Command => {
  return jsonToGenericCommand(data, CommandType.Tensorboard);
};

export const jsonToTensorboards = (data: unknown): Command[] => {
  return jsonToGenericCommands(data, CommandType.Tensorboard);
};

const ioToExperimentConfig = (io: ioTypeExperimentConfig): ExperimentConfig => {
  const config: ExperimentConfig = {
    checkpointPolicy: io.checkpoint_policy,
    checkpointStorage: io.checkpoint_storage ? {
      bucket: io.checkpoint_storage.bucket || undefined,
      hostPath: io.checkpoint_storage.host_path || undefined,
      saveExperimentBest: io.checkpoint_storage.save_experiment_best,
      saveTrialBest: io.checkpoint_storage.save_trial_best,
      saveTrialLatest: io.checkpoint_storage.save_trial_latest,
      storagePath: io.checkpoint_storage.storage_path || undefined,
      type: io.checkpoint_storage.type as CheckpointStorageType || undefined,
    } : undefined,
    dataLayer: io.data_layer ? {
      containerStoragePath: io.data_layer.container_storage_path || undefined,
      type: io.data_layer.type,
    } : undefined,
    description: io.description,
    labels: io.labels || undefined,
    resources: {},
    searcher: {
      ...io.searcher,
      smallerIsBetter: io.searcher.smaller_is_better,
    },
  };
  if (io.resources.max_slots != null) config.resources.maxSlots = io.resources.max_slots;
  return config;
};

export const jsonToExperiment = (data: unknown): ExperimentBase => {
  const io = decode<ioTypeExperiment>(ioExperiment, data);
  return {
    archived: io.archived,
    config: ioToExperimentConfig(io.config),
    configRaw: (data as { config: RawJson }).config,
    endTime: io.end_time || undefined,
    id: io.id,
    progress: io.progress != null ? io.progress : undefined,
    startTime: io.start_time,
    state: io.state as RunState,
    userId: io.owner_id,
  };
};

export const jsonToExperiments = (data: unknown): ExperimentBase[] => {
  const io = decode<ioTypeExperiments>(ioExperiments, data);
  return io.map(jsonToExperiment);
};

const ioToCheckpoint = (io: ioTypeCheckpoint): Checkpoint => {
  return {
    endTime: io.end_time || undefined,
    id: io.id,
    resources: io.resources,
    startTime: io.start_time,
    state: io.state as CheckpointState,
    stepId: io.step_id,
    trialId: io.trial_id,
    uuid: io.uuid || undefined,
    validationMetric: io.validation_metric != null ? io.validation_metric : undefined,
  };
};

const ioToValidationMetrics = (io: ioTypeValidationMetrics): ValidationMetrics => {
  return {
    numInputs: io.num_inputs,
    validationMetrics: dropNonNumericMetrics(io.validation_metrics),
  };
};

const ioToStep = (io: ioTypeStep): Step => {
  return {
    avgMetrics: io.avg_metrics ? dropNonNumericMetrics(io.avg_metrics) : undefined,
    checkpoint: io.checkpoint ? ioToCheckpoint(io.checkpoint) : undefined,
    endTime: io.end_time || undefined,
    id: io.id,
    numBatches: io.num_batches || 0,
    priorBatchesProcessed: io.prior_batches_processed || 0,
    startTime: io.start_time,
    state: io.state as RunState,
    validation: !io.validation ? undefined : {
      endTime: io.validation.end_time || undefined,
      id: io.validation.id,
      metrics: io.validation.metrics != null ?
        ioToValidationMetrics(io.validation.metrics) : undefined,
      startTime: io.validation.start_time,
      state: io.validation.state as RunState,
    },
  };

};

const ioToTrial = (io: ioTypeTrial): TrialItem => {
  return {
    bestAvailableCheckpoint: io.best_available_checkpoint
      ? ioToCheckpoint(io.best_available_checkpoint) : undefined,
    bestValidationMetric: io.best_validation_metric != null ? io.best_validation_metric : undefined,
    endTime: io.end_time || undefined,
    experimentId: io.experiment_id,
    hparams: io.hparams || {},
    id: io.id,
    latestValidationMetrics: io.latest_validation_metrics
      ? ioToValidationMetrics(io.latest_validation_metrics) : undefined,
    numCompletedCheckpoints: io.num_completed_checkpoints,
    numSteps: io.num_steps,
    seed: io.seed,
    startTime: io.start_time,
    state: io.state as RunState,// TODO add checkpoint decoder
    totalBatchesProcessed: io.total_batches_processed || 0,
    url: `/det/trials/${io.id}`,
  };
};

export const jsonToTrialDetails = (data: unknown): TrialDetails => {
  const io = decode<ioTypeTrialDetails>(ioTrialDetails, data);
  return {
    endTime: io.end_time || undefined,
    experimentId: io.experiment_id,
    hparams: io.hparams,
    id: io.id,
    seed: io.seed,
    startTime: io.start_time,
    state: io.state as RunState,
    steps: io.steps.map(ioToStep),
    warmStartCheckpointId: io.warm_start_checkpoint_id != null ?
      io.warm_start_checkpoint_id : undefined,
  };
};

const experimentStateMap = {
  [AB.Determinedexperimentv1State.UNSPECIFIED]: RunState.Unspecified,
  [AB.Determinedexperimentv1State.ACTIVE]: RunState.Active,
  [AB.Determinedexperimentv1State.PAUSED]: RunState.Paused,
  [AB.Determinedexperimentv1State.STOPPINGCANCELED]: RunState.StoppingCanceled,
  [AB.Determinedexperimentv1State.STOPPINGCOMPLETED]: RunState.StoppingCompleted,
  [AB.Determinedexperimentv1State.STOPPINGERROR]: RunState.StoppingError,
  [AB.Determinedexperimentv1State.CANCELED]: RunState.Canceled,
  [AB.Determinedexperimentv1State.COMPLETED]: RunState.Completed,
  [AB.Determinedexperimentv1State.ERROR]: RunState.Errored,
  [AB.Determinedexperimentv1State.DELETED]: RunState.Deleted,
};

export const decodeExperimentState = (data: AB.Determinedexperimentv1State): RunState => {
  return experimentStateMap[data];
};

export const encodeExperimentState = (state: RunState): AB.Determinedexperimentv1State => {
  const stateKey = Object
    .keys(experimentStateMap)
    .find(key => experimentStateMap[key as unknown as AB.Determinedexperimentv1State] === state);
  if (stateKey) return stateKey as unknown as AB.Determinedexperimentv1State;
  return AB.Determinedexperimentv1State.UNSPECIFIED;
};

export const decodeExperimentList = (data: AB.V1Experiment[]): ExperimentItem[] => {
  return data.map(item => ({
    archived: item.archived,
    endTime: item.endTime as unknown as string,
    id: item.id,
    labels: item.labels || [],
    name: item.description,
    numTrials: item.numTrials || 0,
    progress: item.progress != null ? item.progress : undefined,
    startTime: item.startTime as unknown as string,
    state: decodeExperimentState(item.state),
    url: `/det/experiments/${item.id}`,
    username: item.username,
  }));
};

export const jsonToExperimentDetails = (data: unknown): ExperimentDetails => {
  const io = decode<ioTypeExperimentDetails>(ioExperimentDetails, data);
  return {
    archived: io.archived,
    config: ioToExperimentConfig(io.config),
    configRaw: (data as { config: RawJson }).config,
    endTime: io.end_time || undefined,
    id: io.id,
    progress: io.progress != null ? io.progress : undefined,
    startTime: io.start_time,
    state: io.state as RunState,
    trials: io.trials.map(ioToTrial),
    userId: io.owner.id,
    username: io.owner.username,
    validationHistory: io.validation_history.map(vh => ({
      endTime: vh.end_time,
      trialId: vh.trial_id,
      validationError: vh.validation_error != null ? vh.validation_error : undefined,
    })),
  };
};

export const jsonToLogs = (data: unknown): Log[] => {
  const io = decode<ioTypeLogs>(ioLogs, data);
  return io.map(log => ({
    id: log.id,
    level: log.level ? LogLevel[capitalize(log.level) as keyof typeof LogLevel] : undefined,
    message: log.message,
    time: log.time || undefined,
  }));
};

const defaultRegex = /^\[([^\]]+)\]\s([\s\S]*)(\r|\n)$/im;
const kubernetesRegex = /^\s*([0-9a-f]+)\s+(\[[^\]]+\])\s\|\|\s(\S+)\s([\s\S]*)(\r|\n)$/im;

const ioToTrialLog = (io: ioTypeLog): Log => {
  if (defaultRegex.test(io.message)) {
    const matches = io.message.match(defaultRegex) || [];
    const time = matches[1];
    const message = matches[2] || '';
    return { id: io.id, message, time };
  } else if (kubernetesRegex.test(io.message)) {
    const matches = io.message.match(kubernetesRegex) || [];
    const time = matches[3];
    const message = [ matches[1], matches[2], matches[4] ].join(' ');
    return { id: io.id, message, time };
  }
  return { id: io.id, message: io.message };
};

export const jsonToTrialLog = (data: unknown): Log => {
  const io = decode<ioTypeLog>(ioLog, data);
  return ioToTrialLog(io);
};

const ioTaskEventToMessage = (event: string): string => {
  if (defaultRegex.test(event)) {
    const matches = event.match(defaultRegex) || [];
    return matches[2];
  }
  return event;
};

export const jsonToTaskLogs = (data: unknown): Log[] => {
  const io = decode<ioTypeTaskLogs>(ioTaskLogs, data);
  return io
    .filter(log => !log.service_ready_event)
    .map(log => {
      const description = log.snapshot.config.description || '';
      let message = '';
      if (log.scheduled_event) {
        message = `Scheduling ${log.parent_id} (id: ${description})...`;
      } else if (log.assigned_event) {
        message = `${description} was assigned to an agent...`;
      } else if (log.container_started_event) {
        message = `Container of ${description} has started...`;
      } else if (log.terminate_request_event) {
        message = `${description} was requested to terminate...`;
      } else if (log.exited_event) {
        message = `${description} was terminated: ${log.exited_event}`;
      } else if (log.log_event) {
        message = ioTaskEventToMessage(log.log_event);
      }
      return {
        id: log.seq,
        message,
        time: log.time,
      };
    });
};

export const jsonToTrialLogs = (data: unknown): Log[] => {
  const io = decode<ioTypeLogs>(ioLogs, data);
  return io.map(ioToTrialLog);
};
