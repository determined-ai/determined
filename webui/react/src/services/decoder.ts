import dayjs from 'dayjs';

import {
  decode, ioAgents, ioDeterminedInfo, ioExperiment, ioExperimentConfig,
  ioExperimentDetails, ioExperiments, ioGenericCommand, ioGenericCommands, ioLog, ioLogs,
  ioTaskLogs, ioTrialDetails, ioTypeAgents, ioTypeCheckpoint, ioTypeDeterminedInfo,
  ioTypeExperiment, ioTypeExperimentConfig, ioTypeExperimentDetails, ioTypeExperiments,
  ioTypeGenericCommand, ioTypeGenericCommands, ioTypeLatestValidationMetrics,
  ioTypeLog, ioTypeLogs, ioTypeTaskLogs, ioTypeTrial, ioTypeTrialDetails, ioTypeUsers, ioUsers,
} from 'ioTypes';
import {
  Agent, Checkpoint, CheckpointState, CheckpointStorageType, Command, CommandState,
  CommandType, DeterminedInfo, Experiment, ExperimentConfig, ExperimentDetails,
  LatestValidationMetrics, Log, LogLevel, ResourceState, ResourceType, RunState,
  TrialDetails, TrialItem, User,
} from 'types';
import { capitalize } from 'utils/string';

export const jsonToUsers = (data: unknown): User[] => {
  const ioType = decode<ioTypeUsers>(ioUsers, data);
  return ioType.map(user => ({
    id: user.id,
    isActive: user.active,
    isAdmin: user.admin,
    username: user.username,
  }));
};

export const jsonToDeterminedInfo = (data: unknown): DeterminedInfo => {
  const info = decode<ioTypeDeterminedInfo>(ioDeterminedInfo, data);
  return {
    clusterId: info.cluster_id,
    masterId: info.master_id,
    telemetry: {
      enabled: info.telemetry.enabled,
      segmentKey: info.telemetry.segment_key,
    },
    version: info.version,
  };
};

export const jsonToAgents = (data: unknown): Agent[] => {
  const ioType = decode<ioTypeAgents>(ioAgents, data);
  return Object.keys(ioType).map(agentId => {
    const agent = ioType[agentId];
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
  const ioType = decode<ioTypeGenericCommand>(ioGenericCommand, data);
  return {
    config: { ...ioType.config },
    exitStatus: ioType.exit_status || undefined,
    id: ioType.id,
    kind: type,
    misc: ioType.misc ? {
      experimentIds: ioType.misc.experiment_ids || undefined,
      trialIds: ioType.misc.trial_ids || undefined,
    } : undefined,
    owner: {
      id: ioType.owner.id,
      username: ioType.owner.username,
    },
    registeredTime: ioType.registered_time,
    serviceAddress: ioType.service_address || undefined,
    state: ioType.state as CommandState,
  };
};

const jsonToGenericCommands = (data: unknown, type: CommandType): Command[] => {
  const ioType = decode<ioTypeGenericCommands>(ioGenericCommands, data);
  return Object.keys(ioType).map(genericCommandId => {
    return jsonToGenericCommand(ioType[genericCommandId], type);
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

const jsonToExperimentConfig = (data: unknown): ExperimentConfig => {
  const io = decode<ioTypeExperimentConfig>(ioExperimentConfig, data);
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
    labels: io.labels,
    resources: {},
    searcher: {
      ...io.searcher,
      smallerIsBetter: io.searcher.smaller_is_better,
    },
  };
  if (io.resources.max_slots !== undefined)
    config.resources.maxSlots = io.resources.max_slots;
  return config;
};

export const jsonToExperiment = (data: unknown): Experiment => {
  const io = decode<ioTypeExperiment>(ioExperiment, data);
  return {
    archived: io.archived,
    config: jsonToExperimentConfig(io.config),
    /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
    configRaw: (data as any).config,
    endTime: io.end_time || undefined,
    id: io.id,
    ownerId: io.owner_id,
    progress: io.progress !== null ? io.progress : undefined,
    startTime: io.start_time,
    state: io.state as RunState,
  };
};

export const jsonToExperiments = (data: unknown): Experiment[] => {
  const ioType = decode<ioTypeExperiments>(ioExperiments, data);
  return ioType.map(jsonToExperiment);
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
    validationMetric: io.validation_metric !== null ? io.validation_metric : undefined,
  };
};

const ioToLatestValidationMetrics = (
  io: ioTypeLatestValidationMetrics,
): LatestValidationMetrics => {
  return {
    numInputs: io.num_inputs,
    validationMetrics: io.validation_metrics,
  };
};

const ioToTrial = (io: ioTypeTrial): TrialItem => {
  return {
    bestAvailableCheckpoint: io.best_available_checkpoint
      ? ioToCheckpoint(io.best_available_checkpoint) : undefined,
    bestValidationMetric: io.best_validation_metric ? io.best_validation_metric : undefined,
    endTime: io.end_time || undefined,
    experimentId: io.experiment_id,
    hparams: io.hparams || {},
    id: io.id,
    latestValidationMetrics: io.latest_validation_metrics
      ? ioToLatestValidationMetrics(io.latest_validation_metrics) : undefined,
    numBatches: io.num_batches || 0,
    numCompletedCheckpoints: io.num_completed_checkpoints,
    numSteps: io.num_steps,
    seed: io.seed,
    startTime: io.start_time,
    state: io.state as RunState,// TODO add checkpoint decoder
    url: `/ui/trials/${io.id}`,
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
    steps: io.steps.map((step) => ({
      endTime: step.end_time || undefined,
      id: step.id,
      startTime: step.start_time,
      state: step.state as RunState,
    })),
    warmStartCheckpointId: io.warm_start_checkpoint_id !== null ?
      io.warm_start_checkpoint_id : undefined,
  };
};

export const jsonToExperimentDetails = (data: unknown): ExperimentDetails => {
  const ioType = decode<ioTypeExperimentDetails>(ioExperimentDetails, data);
  return {
    archived: ioType.archived,
    config: jsonToExperimentConfig(ioType.config),
    /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
    configRaw: (data as any).config,
    endTime: ioType.end_time || undefined,
    id: ioType.id,
    ownerId: ioType.owner.id,
    progress: ioType.progress !== null ? ioType.progress : undefined,
    startTime: ioType.start_time,
    state: ioType.state as RunState,
    trials: ioType.trials.map(ioToTrial),
    username: ioType.owner.username,
    validationHistory: ioType.validation_history.map(vh => ({
      endTime: vh.end_time,
      trialId: vh.trial_id,
      validationError: vh.validation_error || undefined,
    })),
  };
};

export const jsonToLogs = (data: unknown): Log[] => {
  const ioType = decode<ioTypeLogs>(ioLogs, data);
  return ioType.map(log => ({
    id: log.id,
    level: log.level ? LogLevel[capitalize(log.level) as keyof typeof LogLevel] : undefined,
    message: log.message,
    time: log.time,
  }));
};

const defaultRegex = /^\[([^\]]+)\]\s(.*)$/im;
const kubernetesRegex = /^\s*([0-9a-f]+)\s+(\[[^\]]+\])\s\|\|\s(\S+)\s(.*)$/im;

const ioTrialLogToLog = (io: ioTypeLog): Log => {
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
  const ioType = decode<ioTypeLog>(ioLog, data);
  return ioTrialLogToLog(ioType);
};

const ioTaskEventToMessage = (event: string): string => {
  if (defaultRegex.test(event)) {
    const matches = event.match(defaultRegex) || [];
    return matches[2];
  }
  return event;
};

export const jsonToTaskLogs = (data: unknown): Log[] => {
  const ioType = decode<ioTypeTaskLogs>(ioTaskLogs, data);
  return ioType
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
  const ioType = decode<ioTypeLogs>(ioLogs, data);
  return ioType.map(ioTrialLogToLog);
};
