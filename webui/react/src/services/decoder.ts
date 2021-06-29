import dayjs from 'dayjs';

import * as ioTypes from 'ioTypes';
import * as types from 'types';
import { flattenObject, isNumber, isObject, isPrimitive } from 'utils/data';
import { flattenHyperparameters } from 'utils/experiment';
import { capitalize } from 'utils/string';

import * as Sdk from './api-ts-sdk'; // API Bindings

export const mapV1User = (data: Sdk.V1User): types.DetailedUser => {
  return {
    isActive: data.active,
    isAdmin: data.admin,
    username: data.username,
  };
};

export const mapV1UserList = (data: Sdk.V1GetUsersResponse): types.DetailedUser[] => {
  return (data.users || []).map(user => mapV1User(user));
};

export const jsonToDeterminedInfo = (data: Sdk.V1GetMasterResponse): types.DeterminedInfo => {
  return {
    clusterId: data.clusterId,
    clusterName: data.clusterName,
    isTelemetryEnabled: data.telemetryEnabled === true,
    masterId: data.masterId,
    version: data.version,
  };
};

export const mapV1ResourcePool = (
  data: Sdk.V1ResourcePool,
): types.ResourcePool => {
  return { ...data, slotType: mapV1DeviceType(data.slotType) };
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
        resourceType = mapV1DeviceType(slot.device.type);
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
      resourcePool: agent.resourcePool,
      resources,
    } as types.Agent;
  });
};

const mapV1TaskState =
  (containerState: Sdk.Determinedtaskv1State): types.CommandState => {
    switch (containerState) {
      case Sdk.Determinedtaskv1State.PENDING:
        return types.CommandState.Pending;
      case Sdk.Determinedtaskv1State.ASSIGNED:
        return types.CommandState.Assigned;
      case Sdk.Determinedtaskv1State.PULLING:
        return types.CommandState.Pulling;
      case Sdk.Determinedtaskv1State.STARTING:
        return types.CommandState.Starting;
      case Sdk.Determinedtaskv1State.RUNNING:
        return types.CommandState.Running;
      case Sdk.Determinedtaskv1State.TERMINATED:
        return types.CommandState.Terminated;
      default:
        return types.CommandState.Pending;
    }
  };

const mapCommonV1Task = (
  task: Sdk.V1Command|Sdk.V1Notebook|Sdk.V1Shell|Sdk.V1Tensorboard,
  type: types.CommandType,
): types.CommandTask => {
  return {
    id: task.id,
    name: task.description,
    resourcePool: task.resourcePool,
    startTime: task.startTime as unknown as string,
    state: mapV1TaskState(task.state),
    type,
    username: task.username,
  };
};

export const mapV1Command = (command: Sdk.V1Command): types.CommandTask => {
  return { ...mapCommonV1Task(command, types.CommandType.Command) };
};

export const mapV1Notebook = (notebook: Sdk.V1Notebook): types.CommandTask => {
  return {
    ...mapCommonV1Task(notebook, types.CommandType.Notebook),
    serviceAddress: notebook.serviceAddress,
  };
};

export const mapV1Shell = (shell: Sdk.V1Shell): types.CommandTask => {
  return { ...mapCommonV1Task(shell, types.CommandType.Shell) };
};

export const mapV1Tensorboard =
  (tensorboard: Sdk.V1Tensorboard): types.CommandTask => {
    return {
      ...mapCommonV1Task(tensorboard, types.CommandType.Tensorboard),
      misc: {
        experimentIds: tensorboard.experimentIds || [],
        trialIds: tensorboard.trialIds || [],
      },
      serviceAddress: tensorboard.serviceAddress,
    };
  };

export const mapV1Template = (template: Sdk.V1Template): types.Template => {
  return { config: template.config, name: template.name };
};

const ioToHyperparametereter = (
  io: ioTypes.ioTypeHyperparameter,
): types.Hyperparameter => {
  return {
    base: io.base != null ? io.base : undefined,
    count: io.count != null ? io.count : undefined,
    maxval: io.maxval != null ? io.maxval : undefined,
    minval: io.minval != null ? io.minval : undefined,
    type: io.type as types.HyperparameterType,
    val: io.val != null ? io.val : undefined,
    vals: io.vals != null ? io.vals : undefined,
  };
};

const ioToHyperparametereters = (
  io: ioTypes.ioTypeHyperparameters,
): types.Hyperparameters => {
  const hparams: Record<string, unknown> = {};
  Object.keys(io).forEach(key => {
    /*
     * Keep only the hyperparameters which have a primitive `val` value or
     * where `vals` is a list of primitive values. It is possible for `val`
     * to be anything (map, list, etc). `vals` will either be undefined or
     * a list of anything (same value types as `val`).
     */
    const ioHp = io[key] as ioTypes.ioTypeHyperparameter;
    const valIsPrimitive = isPrimitive(ioHp.val);
    const valListIsPrimitive = Array.isArray(ioHp.vals) && ioHp.vals.reduce((acc, val) => {
      return acc && (val != null && isPrimitive(val));
    }, true);
    if (!ioHp.type && isObject(ioHp)) {
      hparams[key] = ioToHyperparametereters(ioHp as Record<string, unknown>);
    } else if (valIsPrimitive || valListIsPrimitive) {
      hparams[key] = ioToHyperparametereter(ioHp);
    }
  });
  return hparams as types.Hyperparameters;
};

export const ioToExperimentConfig =
(io: ioTypes.ioTypeExperimentConfig): types.ExperimentConfig => {
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
    description: io.description || undefined,
    hyperparameters: ioToHyperparametereters(io.hyperparameters),
    labels: io.labels || undefined,
    name: io.name,
    profiling: { enabled: !!io.profiling?.enabled },
    resources: {},
    searcher: {
      ...io.searcher,
      name: io.searcher.name as types.ExperimentSearcherName,
      smallerIsBetter: io.searcher.smaller_is_better,
    },
  };
  if (io.resources.max_slots != null) config.resources.maxSlots = io.resources.max_slots;
  return config;
};

const checkpointStateMap = {
  [Sdk.Determinedcheckpointv1State.UNSPECIFIED]: types.CheckpointState.Unspecified,
  [Sdk.Determinedcheckpointv1State.ACTIVE]: types.CheckpointState.Active,
  [Sdk.Determinedcheckpointv1State.COMPLETED]: types.CheckpointState.Completed,
  [Sdk.Determinedcheckpointv1State.ERROR]: types.CheckpointState.Error,
  [Sdk.Determinedcheckpointv1State.DELETED]: types.CheckpointState.Deleted,
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
  [Sdk.Determinedexperimentv1State.DELETING]: types.RunState.Deleting,
  [Sdk.Determinedexperimentv1State.DELETEFAILED]: types.RunState.DeleteFailed,
};

export const decodeCheckpointState = (
  data: Sdk.Determinedcheckpointv1State,
): types.CheckpointState => {
  return checkpointStateMap[data];
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

export const decodeGetV1ExperimentRespToExperimentBase = (
  { experiment: exp, config }: Sdk.V1GetExperimentResponse,
): types.ExperimentBase => {
  const ioConfig = ioTypes
    .decode<ioTypes.ioTypeExperimentConfig>(ioTypes.ioExperimentConfig, config);
  const hyperparameters = flattenHyperparameters(
    ioConfig.hyperparameters as types.Hyperparameters,
  );
  return {
    archived: exp.archived,
    config: ioToExperimentConfig(ioConfig),
    configRaw: config,
    description: exp.description,
    endTime: exp.endTime as unknown as string,
    hyperparameters,
    id: exp.id,
    // numTrials
    // labels
    name: exp.name,
    progress: exp.progress != null ? exp.progress : undefined,
    resourcePool: exp.resourcePool || '',
    startTime: exp.startTime as unknown as string,
    state: decodeExperimentState(exp.state),
    username: exp.username,
  };
};

export const mapV1Experiment = (
  data: Sdk.V1Experiment,
): types.ExperimentItem => {
  return {
    archived: data.archived,
    description: data.description,
    endTime: data.endTime as unknown as string,
    id: data.id,
    labels: data.labels || [],
    name: data.name,
    numTrials: data.numTrials || 0,
    progress: data.progress != null ? data.progress : undefined,
    resourcePool: data.resourcePool || '',
    startTime: data.startTime as unknown as string,
    state: decodeExperimentState(data.state),
    username: data.username,
  };
};

export const mapV1ExperimentList = (data: Sdk.V1Experiment[]): types.ExperimentItem[] => {
  return data.map(mapV1Experiment);
};

const filterNonScalarMetrics = (metrics: types.RawJson): types.RawJson | undefined => {
  if (!isObject(metrics)) return undefined;
  const scalarMetrics: types.RawJson = {};
  for (const key in metrics){
    if (isNumber(metrics[key])) {
      scalarMetrics[key] = metrics[key];
    }
  }
  return scalarMetrics;
};

const decodeMetricsWorkload = (data: Sdk.V1MetricsWorkload): types.MetricsWorkload => {
  return {
    endTime: data.endTime as unknown as string,
    metrics: data.metrics ? filterNonScalarMetrics(data.metrics) : undefined,
    startTime: data.startTime as unknown as string,
    totalBatches: data.totalBatches,
  };
};

const decodeCheckpointWorkload = (data: Sdk.V1CheckpointWorkload): types.CheckpointWorkload => {

  const resources: Record<string, number> = {};
  Object.entries(data.resources || {}).forEach(([ res, val ]) => {
    resources[res] = parseFloat(val);
  });

  return {
    endTime: data.endTime as unknown as string,
    resources,
    startTime: data.startTime as unknown as string,
    totalBatches: data.totalBatches,
    uuid: data.uuid,
  };
};

export const decodeCheckpoint = (data: Sdk.V1Checkpoint): types.CheckpointDetail => {
  const resources: Record<string, number> = {};
  Object.entries(data.resources || {}).forEach(([ res, val ]) => {
    resources[res] = parseFloat(val);
  });

  return {
    batch: data.batchNumber,
    endTime: data.endTime && data.endTime as unknown as string,
    experimentId: data.experimentId,
    resources,
    startTime: data.startTime as unknown as string,
    trialId: data.trialId,
    uuid: data.uuid,
    validationMetric: data.searcherMetric,
  };
};

const decodeV1TrialToTrialItem = (data: Sdk.Trialv1Trial): types.TrialItem => {
  return {
    bestAvailableCheckpoint: data.bestCheckpoint && decodeCheckpointWorkload(data.bestCheckpoint),
    bestValidationMetric: data.bestValidation && decodeMetricsWorkload(data.bestValidation),
    endTime: data.endTime && data.endTime as unknown as string,
    experimentId: data.experimentId,
    hyperparameters: flattenObject(data.hparams),
    id: data.id,
    latestValidationMetric: data.latestValidation && decodeMetricsWorkload(data.latestValidation),
    startTime: data.startTime as unknown as string,
    state: decodeExperimentState(data.state),
    totalBatchesProcessed: data.totalBatchesProcessed,
  };
};

export const decodeTrialResponseToTrialDetails = (
  data: Sdk.V1GetTrialResponse,
): types.TrialDetails => {
  const trialItem = decodeV1TrialToTrialItem(data.trial);
  let workloads;

  if (data.workloads) {
    workloads = data.workloads.map(ww => ({
      checkpoint: ww.checkpoint && decodeCheckpointWorkload(ww.checkpoint),
      training: ww.training && decodeMetricsWorkload(ww.training),
      validation: ww.validation && decodeMetricsWorkload(ww.validation),
    }));
  }

  return {
    ...trialItem,
    workloads: workloads || [],
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

const decodeV1LogLevelToLogLevel = (level: Sdk.V1LogLevel): types.LogLevel | undefined => {
  const logLevelMap: Record<Sdk.V1LogLevel, types.LogLevel | undefined> = {
    [Sdk.V1LogLevel.UNSPECIFIED]: undefined,
    [Sdk.V1LogLevel.CRITICAL]: types.LogLevel.Critical,
    [Sdk.V1LogLevel.DEBUG]: types.LogLevel.Debug,
    [Sdk.V1LogLevel.ERROR]: types.LogLevel.Error,
    [Sdk.V1LogLevel.INFO]: types.LogLevel.Info,
    [Sdk.V1LogLevel.TRACE]: types.LogLevel.Trace,
    [Sdk.V1LogLevel.WARNING]: types.LogLevel.Warning,
  };
  return logLevelMap[level];
};

const defaultRegex = /^\[([^\]]+)\]\s([\s\S]*)(\r|\n)$/im;
const kubernetesRegex = /^\s*([0-9a-f]+)\s+(\[[^\]]+\])\s\|\|\s(\S+)\s([\s\S]*)(\r|\n)$/im;

export const jsonToTrialLog = (data: unknown): types.TrialLog => {
  const logData = data as Sdk.V1TrialLogsResponse;
  const log = {
    id: logData.id,
    level: decodeV1LogLevelToLogLevel(logData.level),
    message: logData.message,
    time: logData.timestamp as unknown as string,
  };
  if (defaultRegex.test(logData.message)) {
    const matches = logData.message.match(defaultRegex) || [];
    const message = matches[2] || '';
    log.message = message;
  } else if (kubernetesRegex.test(logData.message)) {
    const matches = logData.message.match(kubernetesRegex) || [];
    const message = [ matches[1], matches[2], matches[4] ].join(' ');
    log.message = message;
  }
  return log;
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

export const mapV1DeviceType = (data: Sdk.Devicev1Type): types.ResourceType => {
  return types.ResourceType[
    data.toString().toUpperCase()
      .replace('TYPE_', '') as keyof typeof types.ResourceType
  ];
};
