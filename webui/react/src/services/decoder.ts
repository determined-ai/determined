import { isNumber } from 'util';

import dayjs from 'dayjs';

import * as ioTypes from 'ioTypes';
import * as types from 'types';
import { capitalize } from 'utils/string';

import * as Sdk from './api-ts-sdk'; // API Bindings
import { LoginResponse } from './types';

const dropNonNumericMetrics = (ioMetrics: ioTypes.ioTypeMetric): Record<string, number> => {
  const metrics: Record<string, number> = {};
  Object.entries(ioMetrics).forEach(([ name, value ]) => {
    if (isNumber(value)) metrics[name] = value;
  });
  return metrics;
};

export const user = (data: Sdk.V1User): types.DetailedUser => {
  return {
    isActive: data.active,
    isAdmin: data.admin,
    username: data.username,
  };
};

export const jsonToUsers = (data: unknown): types.DetailedUser[] => {
  const io = ioTypes.decode<ioTypes.ioTypeDetailedUsers>(ioTypes.ioDetailedUsers, data);
  return io.map(user => ({
    id: user.id,
    isActive: user.active,
    isAdmin: user.admin,
    username: user.username,
  }));
};

export const jsonToLogin = (data: unknown): LoginResponse => {
  const io = ioTypes.decode<ioTypes.ioTypeLogin>(ioTypes.ioLogin, data);
  return { token: io.token };
};

export const jsonToDeterminedInfo = (data: unknown): types.DeterminedInfo => {
  const io = ioTypes.decode<ioTypes.ioTypeDeterminedInfo>(ioTypes.ioDeterminedInfo, data);
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

export const jsonToAgents = (agents: Array<Sdk.V1Agent>): types.Agent[] => {
  return agents.map(agent => {
    const agentSlots = agent.slots || {};
    const resources = Object.keys(agentSlots).map(slotId => {
      const slot = agentSlots[slotId];

      let resourceContainer = undefined;
      if (slot.container) {
        let resourceContainerState = undefined;
        if (slot.container.state) {
          resourceContainerState = types.ResourceState[
            capitalize(
              slot.container.state.toString().replace('STATE_', ''),
            ) as keyof typeof types.ResourceState
          ];
        }

        resourceContainer = {
          id: slot.container.id,
          state: resourceContainerState,
        };
      }

      let resourceType = types.ResourceType.UNSPECIFIED;
      if (slot.device?.type) {
        resourceType = types.ResourceType[
          slot.device.type.toString().toUpperCase()
            .replace('TYPE_', '') as keyof typeof types.ResourceType
        ];
      }

      return {
        container: resourceContainer,
        enabled: slot.enabled,
        id: slot.id,
        name: slot.device?.brand,
        type: resourceType,
        uuid: slot.device?.uuid || undefined,
      };
    });

    return {
      id: agent.id,
      registeredTime: dayjs(agent.registeredTime).unix(),
      resources,
    } as types.Agent;
  });
};

export const jsonToGenericCommand = (data: unknown, type: types.CommandType): types.Command => {
  const io = ioTypes.decode<ioTypes.ioTypeGenericCommand>(ioTypes.ioGenericCommand, data);
  const command: types.Command = {
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
    state: io.state as types.CommandState,
    user: { username: io.owner.username },
  };
  return command as types.Command;
};

const jsonToGenericCommands = (data: unknown, type: types.CommandType): types.Command[] => {
  const io = ioTypes.decode<ioTypes.ioTypeGenericCommands>(ioTypes.ioGenericCommands, data);
  return Object.keys(io).map(genericCommandId => {
    return jsonToGenericCommand(io[genericCommandId], type);
  });
};

export const jsonToCommands = (data: unknown): types.Command[] => {
  return jsonToGenericCommands(data, types.CommandType.Command);
};

export const jsonToNotebook = (data: unknown): types.Command => {
  return jsonToGenericCommand(data, types.CommandType.Notebook);
};

export const jsonToNotebooks = (data: unknown): types.Command[] => {
  return jsonToGenericCommands(data, types.CommandType.Notebook);
};

export const jsonToShells = (data: unknown): types.Command[] => {
  return jsonToGenericCommands(data, types.CommandType.Shell);
};

export const jsonToTensorboard = (data: unknown): types.Command => {
  return jsonToGenericCommand(data, types.CommandType.Tensorboard);
};

export const jsonToTensorboards = (data: unknown): types.Command[] => {
  return jsonToGenericCommands(data, types.CommandType.Tensorboard);
};

const ioToExperimentConfig = (io: ioTypes.ioTypeExperimentConfig): types.ExperimentConfig => {
  const config: types.ExperimentConfig = {
    checkpointPolicy: io.checkpoint_policy,
    checkpointStorage: io.checkpoint_storage ? {
      bucket: io.checkpoint_storage.bucket || undefined,
      hostPath: io.checkpoint_storage.host_path || undefined,
      saveExperimentBest: io.checkpoint_storage.save_experiment_best,
      saveTrialBest: io.checkpoint_storage.save_trial_best,
      saveTrialLatest: io.checkpoint_storage.save_trial_latest,
      storagePath: io.checkpoint_storage.storage_path || undefined,
      type: io.checkpoint_storage.type as types.CheckpointStorageType || undefined,
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

export const jsonToExperiment = (data: unknown): types.ExperimentBase => {
  const io = ioTypes.decode<ioTypes.ioTypeExperiment>(ioTypes.ioExperiment, data);
  return {
    archived: io.archived,
    config: ioToExperimentConfig(io.config),
    configRaw: (data as { config: types.RawJson }).config,
    endTime: io.end_time || undefined,
    id: io.id,
    progress: io.progress != null ? io.progress : undefined,
    startTime: io.start_time,
    state: io.state as types.RunState,
    userId: io.owner_id,
  };
};

export const jsonToExperiments = (data: unknown): types.ExperimentBase[] => {
  const io = ioTypes.decode<ioTypes.ioTypeExperiments>(ioTypes.ioExperiments, data);
  return io.map(jsonToExperiment);
};

const ioToCheckpoint = (io: ioTypes.ioTypeCheckpoint): types.Checkpoint => {
  return {
    endTime: io.end_time || undefined,
    id: io.id,
    resources: io.resources,
    startTime: io.start_time,
    state: io.state as types.CheckpointState,
    stepId: io.step_id,
    trialId: io.trial_id,
    uuid: io.uuid || undefined,
    validationMetric: io.validation_metric != null ? io.validation_metric : undefined,
  };
};

const ioToValidationMetrics = (io: ioTypes.ioTypeValidationMetrics): types.ValidationMetrics => {
  return {
    numInputs: io.num_inputs,
    validationMetrics: dropNonNumericMetrics(io.validation_metrics),
  };
};

const ioToStep = (io: ioTypes.ioTypeStep): types.Step => {
  return {
    avgMetrics: io.avg_metrics ? dropNonNumericMetrics(io.avg_metrics) : undefined,
    checkpoint: io.checkpoint ? ioToCheckpoint(io.checkpoint) : undefined,
    endTime: io.end_time || undefined,
    id: io.id,
    numBatches: io.num_batches || 0,
    priorBatchesProcessed: io.prior_batches_processed || 0,
    startTime: io.start_time,
    state: io.state as types.RunState,
    validation: !io.validation ? undefined : {
      endTime: io.validation.end_time || undefined,
      id: io.validation.id,
      metrics: io.validation.metrics != null ?
        ioToValidationMetrics(io.validation.metrics) : undefined,
      startTime: io.validation.start_time,
      state: io.validation.state as types.RunState,
    },
  };

};

const ioToTrial = (io: ioTypes.ioTypeTrial): types.TrialItem => {
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
    state: io.state as types.RunState,// TODO add checkpoint decoder
    totalBatchesProcessed: io.total_batches_processed || 0,
    url: `/experiments/${io.experiment_id}/trials/${io.id}`,
  };
};

export const jsonToTrialDetails = (data: unknown): types.TrialDetails => {
  const io = ioTypes.decode<ioTypes.ioTypeTrialDetails>(ioTypes.ioTrialDetails, data);
  return {
    endTime: io.end_time || undefined,
    experimentId: io.experiment_id,
    hparams: io.hparams,
    id: io.id,
    seed: io.seed,
    startTime: io.start_time,
    state: io.state as types.RunState,
    steps: io.steps.map(ioToStep),
    warmStartCheckpointId: io.warm_start_checkpoint_id != null ?
      io.warm_start_checkpoint_id : undefined,
  };
};

const experimentStateMap = {
  [Sdk.Determinedexperimentv1State.UNSPECIFIED]: types.RunState.Unspecified,
  [Sdk.Determinedexperimentv1State.ACTIVE]: types.RunState.Active,
  [Sdk.Determinedexperimentv1State.PAUSED]: types.RunState.Paused,
  [Sdk.Determinedexperimentv1State.STOPPINGCANCELED]: types.RunState.StoppingCanceled,
  [Sdk.Determinedexperimentv1State.STOPPINGCOMPLETED]: types.RunState.StoppingCompleted,
  [Sdk.Determinedexperimentv1State.STOPPINGERROR]: types.RunState.StoppingError,
  [Sdk.Determinedexperimentv1State.CANCELED]: types.RunState.Canceled,
  [Sdk.Determinedexperimentv1State.COMPLETED]: types.RunState.Completed,
  [Sdk.Determinedexperimentv1State.ERROR]: types.RunState.Errored,
  [Sdk.Determinedexperimentv1State.DELETED]: types.RunState.Deleted,
};

export const decodeExperimentState = (data: Sdk.Determinedexperimentv1State): types.RunState => {
  return experimentStateMap[data];
};

export const encodeExperimentState = (state: types.RunState): Sdk.Determinedexperimentv1State => {
  const stateKey = Object
    .keys(experimentStateMap)
    .find(key => experimentStateMap[key as unknown as Sdk.Determinedexperimentv1State] === state);
  if (stateKey) return stateKey as unknown as Sdk.Determinedexperimentv1State;
  return Sdk.Determinedexperimentv1State.UNSPECIFIED;
};

export const decodeExperimentList = (data: Sdk.V1Experiment[]): types.ExperimentItem[] => {
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
    url: `/experiments/${item.id}`,
    username: item.username,
  }));
};

export const jsonToExperimentDetails = (data: unknown): types.ExperimentDetails => {
  const io = ioTypes.decode<ioTypes.ioTypeExperimentDetails>(ioTypes.ioExperimentDetails, data);
  return {
    archived: io.archived,
    config: ioToExperimentConfig(io.config),
    configRaw: (data as { config: types.RawJson }).config,
    endTime: io.end_time || undefined,
    id: io.id,
    progress: io.progress != null ? io.progress : undefined,
    startTime: io.start_time,
    state: io.state as types.RunState,
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

export const jsonToLogs = (data: unknown): types.Log[] => {
  const io = ioTypes.decode<ioTypes.ioTypeLogs>(ioTypes.ioLogs, data);
  return io.map(log => ({
    id: log.id,
    level: log.level ?
      types.LogLevel[capitalize(log.level) as keyof typeof types.LogLevel] : undefined,
    message: log.message,
    time: log.time || undefined,
  }));
};

const defaultRegex = /^\[([^\]]+)\]\s([\s\S]*)(\r|\n)$/im;
const kubernetesRegex = /^\s*([0-9a-f]+)\s+(\[[^\]]+\])\s\|\|\s(\S+)\s([\s\S]*)(\r|\n)$/im;

const ioToTrialLog = (io: ioTypes.ioTypeLog): types.Log => {
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

export const jsonToTrialLog = (data: unknown): types.Log => {
  const io = ioTypes.decode<ioTypes.ioTypeLog>(ioTypes.ioLog, data);
  return ioToTrialLog(io);
};

const ioTaskEventToMessage = (event: string): string => {
  if (defaultRegex.test(event)) {
    const matches = event.match(defaultRegex) || [];
    return matches[2];
  }
  return event;
};

export const jsonToTaskLogs = (data: unknown): types.Log[] => {
  const io = ioTypes.decode<ioTypes.ioTypeTaskLogs>(ioTypes.ioTaskLogs, data);
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

export const jsonToTrialLogs = (data: unknown): types.Log[] => {
  const io = ioTypes.decode<ioTypes.ioTypeLogs>(ioTypes.ioLogs, data);
  return io.map(ioToTrialLog);
};
