import dayjs from 'dayjs';

import * as ioTypes from 'ioTypes';
import * as types from 'types';
import { flattenObject, isNullOrUndefined, isNumber, isObject, isPrimitive } from 'utils/data';
import { capitalize } from 'utils/string';

import * as Sdk from './api-ts-sdk'; // API Bindings

export const mapV1User = (data: Sdk.V1User): types.DetailedUser => {
  return {
    displayName: data.displayName,
    id: data.id,
    isActive: data.active,
    isAdmin: data.admin,
    username: data.username,
  };
};

export const mapV1UserList = (data: Sdk.V1GetUsersResponse): types.DetailedUser[] => {
  return (data.users || []).map(user => mapV1User(user));
};

export const mapV1Pagination = (data: Sdk.V1Pagination): types.Pagination => {
  return {
    limit: data.limit ?? 0,
    offset: data.offset ?? 0,
  };
};

export const mapV1MasterInfo = (data: Sdk.V1GetMasterResponse): types.DeterminedInfo => {
  // Validate branding against `BrandingType` enum.
  const branding = Object.values(types.BrandingType).reduce((acc, value) => {
    if (value === data.branding) acc = data.branding;
    return acc;
  }, types.BrandingType.Determined);

  return {
    branding,
    checked: true,
    clusterId: data.clusterId,
    clusterName: data.clusterName,
    externalLoginUri: data.externalLoginUri,
    externalLogoutUri: data.externalLogoutUri,
    isTelemetryEnabled: data.telemetryEnabled === true,
    masterId: data.masterId,
    ssoProviders: data.ssoProviders,
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
  task: Sdk.V1Command | Sdk.V1Notebook | Sdk.V1Shell | Sdk.V1Tensorboard,
  type: types.CommandType,
): types.CommandTask => {
  return {
    displayName: task.displayName || '',
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
    ...mapCommonV1Task(notebook, types.CommandType.JupyterLab),
    serviceAddress: notebook.serviceAddress,
  };
};

export const mapV1Shell = (shell: Sdk.V1Shell): types.CommandTask => {
  return { ...mapCommonV1Task(shell, types.CommandType.Shell) };
};

export const mapV1TensorBoard =
  (tensorboard: Sdk.V1Tensorboard): types.CommandTask => {
    return {
      ...mapCommonV1Task(tensorboard, types.CommandType.TensorBoard),
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

export const mapV1Task = (task: Sdk.V1Task): types.TaskItem => {
  return {
    allocations: task.allocations?.map(a => {
      const setState = {
        STATE_ASSIGNED: types.CommandState.Assigned,
        STATE_PENDING: types.CommandState.Pending,
        STATE_PULLING: types.CommandState.Pulling,
        STATE_RUNNING: types.CommandState.Running,
        STATE_STARTING: types.CommandState.Starting,
        STATE_TERMINATED: types.CommandState.Terminated,
        STATE_TERMINATING: types.CommandState.Terminating,
      }[String(a?.state) || 'STATE_PENDING'] || types.CommandState.Pending;

      return {
        isReady: a.isReady || false,
        state: setState,
        taskId: a.taskId,
      };
    }) || [],
    taskId: task.taskId || '',
  };
};

export const mapV1Model = (model: Sdk.V1Model): types.ModelItem => {
  return {
    archived: model.archived,
    creationTime: model.creationTime as unknown as string,
    description: model.description,
    id: model.id,
    labels: model.labels,
    lastUpdatedTime: model.lastUpdatedTime as unknown as string,
    metadata: model.metadata,
    name: model.name,
    notes: model.notes,
    numVersions: model.numVersions,
    username: model.username,
  };
};

export const mapV1ModelVersion = (
  modelVersion: Sdk.V1ModelVersion,
): types.ModelVersion => {
  return {
    checkpoint: decodeCheckpoint(modelVersion.checkpoint),
    comment: modelVersion.comment,
    creationTime: modelVersion.creationTime as unknown as string,
    id: modelVersion.id,
    labels: modelVersion.labels,
    lastUpdatedTime: modelVersion.lastUpdatedTime as unknown as string,
    metadata: modelVersion.metadata,
    model: mapV1Model(modelVersion.model),
    name: modelVersion.name,
    notes: modelVersion.notes,
    username: modelVersion.username,
    version: modelVersion.version,
  };
};

export const mapV1ModelDetails = (
  modelDetailsResponse: Sdk.V1GetModelVersionsResponse,
): types.ModelVersions | undefined => {
  if (!modelDetailsResponse.model ||
    !modelDetailsResponse.modelVersions ||
    !modelDetailsResponse.pagination) return;
  return {
    model: mapV1Model(modelDetailsResponse.model),
    modelVersions: modelDetailsResponse.modelVersions.map(version =>
      mapV1ModelVersion(version) as types.ModelVersion),
    pagination: modelDetailsResponse.pagination,
  };
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
      return acc && (isPrimitive(val) && !isNullOrUndefined(val));
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
  [Sdk.Determinedexperimentv1State.STOPPINGKILLED]: types.RunState.StoppingCanceled,
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

export const mapV1GetExperimentResponse = (
  { experiment: exp, config, jobSummary }: Sdk.V1GetExperimentResponse,
): types.ExperimentBase => {
  const ioConfig = ioTypes
    .decode<ioTypes.ioTypeExperimentConfig>(ioTypes.ioExperimentConfig, config);
  const continueFn = (value: unknown) => !(value as types.HyperparameterBase).type;
  const hyperparameters = flattenObject<types.HyperparameterBase>(
    ioConfig.hyperparameters,
    { continueFn },
  ) as types.HyperparametersFlattened;
  const v1Exp = mapV1Experiment(exp);
  v1Exp.jobSummary = jobSummary;
  const resolvedState = v1Exp.state === types.RunState.Active && v1Exp.jobSummary ?
    v1Exp.jobSummary.state : v1Exp.state;
  v1Exp.state = resolvedState;

  return {
    ...v1Exp,
    config: ioToExperimentConfig(ioConfig),
    configRaw: config,
    hyperparameters,
  };
};

export const mapV1Experiment = (
  data: Sdk.V1Experiment,
): types.ExperimentItem => {
  return {
    archived: data.archived,
    description: data.description,
    endTime: data.endTime as unknown as string,
    forkedFrom: data.forkedFrom,
    id: data.id,
    jobId: data.jobId,
    labels: data.labels || [],
    name: data.name,
    notes: data.notes,
    numTrials: data.numTrials || 0,
    progress: data.progress != null ? data.progress : undefined,
    resourcePool: data.resourcePool || '',
    searcherType: data.searcherType,
    startTime: data.startTime as unknown as string,
    state: decodeExperimentState(data.state),
    trialIds: data.trialIds || [],
    username: data.username,
  };
};

export const mapV1ExperimentList = (data: Sdk.V1Experiment[]): types.ExperimentItem[] => {
  return data.map(mapV1Experiment);
};

const filterNonScalarMetrics = (metrics: types.RawJson): types.RawJson | undefined => {
  if (!isObject(metrics)) return undefined;
  const scalarMetrics: types.RawJson = {};
  for (const key in metrics) {
    if ([ 'Infinity', '-Infinity', 'NaN' ].includes(metrics[key])) {
      scalarMetrics[key] = Number(metrics[key]);
    } else if (isNumber(metrics[key])) {
      scalarMetrics[key] = metrics[key];
    }
  }
  return scalarMetrics;
};

const decodeMetricsWorkload = (data: Sdk.V1MetricsWorkload): types.MetricsWorkload => {
  return {
    endTime: data.endTime as unknown as string,
    metrics: data.metrics ? filterNonScalarMetrics(data.metrics) : undefined,
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
    state: decodeCheckpointState(data.state),
    totalBatches: data.totalBatches,
    uuid: data.uuid,
  };
};

const decodeValidationMetrics = (data: Sdk.V1Metrics): types.Metrics => {
  return { validationMetrics: data.avgMetrics };
};

export const decodeCheckpoint = (data: Sdk.V1Checkpoint): types.CheckpointDetail => {
  const resources: Record<string, number> = {};
  Object.entries(data.resources || {}).forEach(([ res, val ]) => {
    resources[res] = parseFloat(val);
  });

  // TODO @emily the following has been brainlessly changed to compile

  return {
    batch: data.metadata['latest_batch'],
    endTime: data.reportTime && data.reportTime as unknown as string,
    experimentId: data.training.experimentId,
    metrics: data.training.validationMetrics ? decodeValidationMetrics(
      data.training.validationMetrics,
    ) : undefined,
    resources,
    state: decodeCheckpointState(data.state || Sdk.Determinedcheckpointv1State.UNSPECIFIED),
    trialId: data.training.trialId || -1, // TODO maybe it becomes required again
    uuid: data.uuid,
    validationMetric: data.training.searcherMetric,
  };
};

export const decodeV1TrialToTrialItem = (data: Sdk.Trialv1Trial): types.TrialItem => {
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

  const EMPTY_STATES = new Set([ 'UNSPECIFIED', '', undefined ]);

  return {
    ...trialItem,
    runnerState: EMPTY_STATES.has(data.trial.runnerState) ? undefined : data.trial.runnerState,
    workloads: workloads || [],
  };
};

export const jsonToClusterLog = (data: unknown): types.Log => {
  const logData = data as Sdk.V1MasterLogsResponse;
  return ({
    id: logData.logEntry?.id ?? 0,
    level: decodeV1LogLevelToLogLevel(logData.logEntry?.level ?? Sdk.V1LogLevel.UNSPECIFIED),
    message: logData.logEntry?.message ?? '',
    time: logData.logEntry?.timestamp as unknown as string,
  });
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

export const jsonToTaskLog = (data: unknown): types.Log => {
  const logData = data as Sdk.V1TaskLogsResponse;
  return ({
    id: logData.id,
    level: decodeV1LogLevelToLogLevel(logData.level ?? Sdk.V1LogLevel.UNSPECIFIED),
    message: (logData.message ?? '').trim(),      // Task logs comes with tailing `\n`.
    time: logData.timestamp as unknown as string,
  });
};

export const mapV1DeviceType = (data: Sdk.Determineddevicev1Type): types.ResourceType => {
  return types.ResourceType[
    data.toString().toUpperCase()
      .replace('TYPE_', '') as keyof typeof types.ResourceType
  ];
};
