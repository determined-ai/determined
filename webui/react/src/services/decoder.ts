import dayjs from 'dayjs';

import * as ioTypes from 'ioTypes';
import { Pagination, RawJson } from 'shared/types';
import {
  flattenObject,
  isNullOrUndefined,
  isNumber,
  isObject,
  isPrimitive,
} from 'shared/utils/data';
import { capitalize } from 'shared/utils/string';
import { BrandingType, DeterminedInfo } from 'stores/determinedInfo';
import * as types from 'types';

import * as Sdk from './api-ts-sdk'; // API Bindings

export const mapV1User = (data: Sdk.V1User): types.DetailedUser => {
  return {
    agentUserGroup: data.agentUserGroup,
    displayName: data.displayName,
    id: data.id || 0,
    isActive: data.active,
    isAdmin: data.admin,
    modifiedAt: new Date(data.modifiedAt || 1).getTime(),
    username: data.username,
  };
};

export const mapV1UserList = (data: Sdk.V1GetUsersResponse): types.DetailedUser[] => {
  return (data.users || []).map((user) => mapV1User(user));
};

export const mapV1Role = (role: Sdk.V1Role): types.UserRole => {
  return {
    id: role.roleId,
    name: role.name || '',
    permissions: (role.permissions || []).map(mapV1Permission),
  };
};

export const mapV1UserRole = (res: Sdk.V1RoleWithAssignments): types.UserRole => {
  const { role, userRoleAssignments } = res;
  return {
    fromUser:
      (userRoleAssignments?.filter((u) => !!(u.userId && u.roleAssignment.scopeCluster)) || [])
        .length > 0,
    id: role?.roleId || 0,
    name: role?.name || '',
    permissions: (role?.permissions || []).map(mapV1Permission),
  };
};

export const mapV1GroupRole = (res: Sdk.V1GetRolesAssignedToGroupResponse): types.UserRole[] => {
  const { roles, assignments } = res;
  return roles.map((role) => ({
    id: role.roleId,
    name: role.name || '',
    permissions: (role.permissions || []).map(mapV1Permission),
    scopeCluster: assignments.find((a) => a.roleId === role.roleId)?.scopeCluster,
  }));
};

export const mapV1Permission = (permission: Sdk.V1Permission): types.Permission => {
  return {
    id: permission.id,
    scopeCluster: permission.scopeTypeMask?.cluster || false,
    scopeWorkspace: permission.scopeTypeMask?.workspace || false,
  };
};

export const mapV1UserAssignment = (
  assignment: Sdk.V1RoleAssignmentSummary,
): types.UserAssignment => {
  return {
    roleId: assignment.roleId,
    scopeCluster: assignment.scopeCluster || false,
    workspaces: assignment.scopeWorkspaceIds || [],
  };
};

export const mapV1Pagination = (data?: Sdk.V1Pagination): Pagination => {
  return {
    limit: data?.limit ?? 0,
    offset: data?.offset ?? 0,
    total: data?.total ?? 0,
  };
};

export const mapV1MasterInfo = (data: Sdk.V1GetMasterResponse): DeterminedInfo => {
  // Validate branding against `BrandingType` enum.
  const branding = Object.values(BrandingType).reduce((acc, value) => {
    if (value === data.branding) acc = data.branding;
    return acc;
  }, BrandingType.Determined);

  return {
    branding,
    checked: true,
    clusterId: data.clusterId,
    clusterName: data.clusterName,
    externalLoginUri: data.externalLoginUri,
    externalLogoutUri: data.externalLogoutUri,
    featureSwitches: data.featureSwitches || [],
    isTelemetryEnabled: data.telemetryEnabled === true,
    masterId: data.masterId,
    rbacEnabled: !!data.rbacEnabled,
    ssoProviders: data.ssoProviders,
    userManagementEnabled: !!data.userManagementEnabled,
    version: data.version,
  };
};

export const mapV1ResourcePool = (data: Sdk.V1ResourcePool): types.ResourcePool => {
  return { ...data, slotType: mapV1DeviceType(data.slotType) };
};

export const jsonToAgents = (agents: Array<Sdk.V1Agent>): types.Agent[] => {
  return agents.map((agent) => {
    const agentSlots = agent.slots || {};
    const resources = Object.keys(agentSlots).map((slotId) => {
      const slot = agentSlots[slotId];

      let resourceContainer:
        | {
            id: string;
            state: types.ResourceState | undefined;
          }
        | undefined = undefined;
      if (slot.container) {
        let resourceContainerState: types.ResourceState | undefined = undefined;
        if (slot.container.state) {
          resourceContainerState =
            types.ResourceState[
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

      let resourceType: types.ResourceType = types.ResourceType.UNSPECIFIED;
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
      resourcePools: agent.resourcePools,
      resources,
    } as types.Agent;
  });
};

const mapV1TaskState = (containerState: Sdk.Taskv1State): types.CommandState => {
  switch (containerState) {
    case Sdk.Taskv1State.PULLING:
      return types.CommandState.Pulling;
    case Sdk.Taskv1State.STARTING:
      return types.CommandState.Starting;
    case Sdk.Taskv1State.RUNNING:
      return types.CommandState.Running;
    case Sdk.Taskv1State.TERMINATED:
      return types.CommandState.Terminated;
    default:
      return types.CommandState.Queued;
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
    userId: task.userId ?? 0,
    workspaceId: task.workspaceId,
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

export const mapV1TensorBoard = (tensorboard: Sdk.V1Tensorboard): types.CommandTask => {
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
    allocations:
      task.allocations?.map((a) => {
        const setState =
          {
            STATE_PULLING: types.CommandState.Pulling,
            STATE_RUNNING: types.CommandState.Running,
            STATE_STARTING: types.CommandState.Starting,
            STATE_TERMINATED: types.CommandState.Terminated,
            STATE_TERMINATING: types.CommandState.Terminating,
            STATE_WAITING: types.CommandState.Waiting,
          }[String(a?.state) || 'STATE_QUEUED'] || types.CommandState.Queued;

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
    userId: model.userId ?? 0,
    workspaceId: model.workspaceId,
  };
};

export const mapV1ModelVersion = (modelVersion: Sdk.V1ModelVersion): types.ModelVersion => {
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
    userId: modelVersion.userId ?? 0,
    version: modelVersion.version,
  };
};

export const mapV1ModelDetails = (
  modelDetailsResponse: Sdk.V1GetModelVersionsResponse,
): types.ModelVersions | undefined => {
  if (
    !modelDetailsResponse.model ||
    !modelDetailsResponse.modelVersions ||
    !modelDetailsResponse.pagination
  )
    return;
  return {
    model: mapV1Model(modelDetailsResponse.model),
    modelVersions: modelDetailsResponse.modelVersions.map(
      (version) => mapV1ModelVersion(version) as types.ModelVersion,
    ),
    pagination: modelDetailsResponse.pagination,
  };
};

const ioToHyperparametereter = (io: ioTypes.ioTypeHyperparameter): types.Hyperparameter => {
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

const ioToHyperparametereters = (io: ioTypes.ioTypeHyperparameters): types.Hyperparameters => {
  const hparams: Record<string, unknown> = {};
  Object.keys(io).forEach((key) => {
    /*
     * Keep only the hyperparameters which have a primitive `val` value or
     * where `vals` is a list of primitive values. It is possible for `val`
     * to be anything (map, list, etc). `vals` will either be undefined or
     * a list of anything (same value types as `val`).
     */
    const ioHp = io[key] as ioTypes.ioTypeHyperparameter;
    const valIsPrimitive = isPrimitive(ioHp.val);
    const valListIsPrimitive =
      Array.isArray(ioHp.vals) &&
      ioHp.vals.reduce((acc, val) => {
        return acc && isPrimitive(val) && !isNullOrUndefined(val);
      }, true);
    if (!ioHp.type && isObject(ioHp)) {
      hparams[key] = ioToHyperparametereters(ioHp as Record<string, unknown>);
    } else if (valIsPrimitive || valListIsPrimitive) {
      hparams[key] = ioToHyperparametereter(ioHp);
    }
  });
  return hparams as types.Hyperparameters;
};

export const ioToExperimentConfig = (
  io: ioTypes.ioTypeExperimentConfig,
): types.ExperimentConfig => {
  const config: types.ExperimentConfig = {
    checkpointPolicy: io.checkpoint_policy,
    checkpointStorage: io.checkpoint_storage
      ? {
          bucket: io.checkpoint_storage.bucket || undefined,
          hostPath: io.checkpoint_storage.host_path || undefined,
          saveExperimentBest: io.checkpoint_storage.save_experiment_best,
          saveTrialBest: io.checkpoint_storage.save_trial_best,
          saveTrialLatest: io.checkpoint_storage.save_trial_latest,
          storagePath: io.checkpoint_storage.storage_path || undefined,
          type: (io.checkpoint_storage.type as types.CheckpointStorageType) || undefined,
        }
      : undefined,
    description: io.description || undefined,
    hyperparameters: ioToHyperparametereters(io.hyperparameters),
    labels: io.labels || undefined,
    maxRestarts: io.max_restarts,
    name: io.name,
    profiling: { enabled: !!io.profiling?.enabled },
    resources: {},
    searcher: {
      ...io.searcher,
      name: io.searcher.name as types.ExperimentSearcherName,
      smallerIsBetter: io.searcher.smaller_is_better,
      sourceTrialId: io.searcher.source_trial_id ?? undefined,
    },
  };
  if (io.resources.max_slots != null) config.resources.maxSlots = io.resources.max_slots;
  return config;
};

const checkpointStateMap = {
  [Sdk.Checkpointv1State.UNSPECIFIED]: types.CheckpointState.Unspecified,
  [Sdk.Checkpointv1State.ACTIVE]: types.CheckpointState.Active,
  [Sdk.Checkpointv1State.COMPLETED]: types.CheckpointState.Completed,
  [Sdk.Checkpointv1State.ERROR]: types.CheckpointState.Error,
  [Sdk.Checkpointv1State.DELETED]: types.CheckpointState.Deleted,
};

const experimentStateMap = {
  [Sdk.Experimentv1State.UNSPECIFIED]: types.RunState.Unspecified,
  [Sdk.Experimentv1State.ACTIVE]: types.RunState.Active,
  [Sdk.Experimentv1State.PAUSED]: types.RunState.Paused,
  [Sdk.Experimentv1State.STOPPINGCANCELED]: types.RunState.StoppingCanceled,
  [Sdk.Experimentv1State.STOPPINGCOMPLETED]: types.RunState.StoppingCompleted,
  [Sdk.Experimentv1State.STOPPINGERROR]: types.RunState.StoppingError,
  [Sdk.Experimentv1State.CANCELED]: types.RunState.Canceled,
  [Sdk.Experimentv1State.COMPLETED]: types.RunState.Completed,
  [Sdk.Experimentv1State.ERROR]: types.RunState.Error,
  [Sdk.Experimentv1State.DELETED]: types.RunState.Deleted,
  [Sdk.Experimentv1State.DELETING]: types.RunState.Deleting,
  [Sdk.Experimentv1State.DELETEFAILED]: types.RunState.DeleteFailed,
  [Sdk.Experimentv1State.STOPPINGKILLED]: types.RunState.StoppingCanceled,
  [Sdk.Experimentv1State.QUEUED]: types.RunState.Queued,
  [Sdk.Experimentv1State.PULLING]: types.RunState.Pulling,
  [Sdk.Experimentv1State.STARTING]: types.RunState.Starting,
  [Sdk.Experimentv1State.RUNNING]: types.RunState.Running,
};

export const decodeCheckpointState = (data: Sdk.Checkpointv1State): types.CheckpointState => {
  return checkpointStateMap[data];
};

export const encodeCheckpointState = (state: types.CheckpointState): Sdk.Checkpointv1State => {
  const stateKey = Object.keys(checkpointStateMap).find(
    (key) => checkpointStateMap[key as unknown as Sdk.Checkpointv1State] === state,
  );
  if (stateKey) return stateKey as unknown as Sdk.Checkpointv1State;
  return Sdk.Checkpointv1State.UNSPECIFIED;
};

export const decodeExperimentState = (data: Sdk.Experimentv1State): types.RunState => {
  return experimentStateMap[data];
};

export const encodeExperimentState = (state: types.RunState): Sdk.Experimentv1State => {
  const stateKey = Object.keys(experimentStateMap).find(
    (key) => experimentStateMap[key as unknown as Sdk.Experimentv1State] === state,
  );
  if (stateKey) return stateKey as unknown as Sdk.Experimentv1State;
  return Sdk.Experimentv1State.UNSPECIFIED;
};

export const mapV1GetExperimentDetailsResponse = ({
  experiment: exp,
  jobSummary,
}: Sdk.V1GetExperimentResponse): types.ExperimentBase => {
  const ioConfig = ioTypes.decode<ioTypes.ioTypeExperimentConfig>(
    ioTypes.ioExperimentConfig,
    exp.config,
  );
  const continueFn = (value: unknown) => !(value as types.HyperparameterBase).type;
  const hyperparameters = flattenObject<types.HyperparameterBase>(ioConfig.hyperparameters, {
    continueFn,
  }) as types.HyperparametersFlattened;
  const v1Exp = mapV1Experiment(exp, jobSummary);
  return {
    ...v1Exp,
    config: ioToExperimentConfig(ioConfig),
    configRaw: exp.config,
    hyperparameters,
    originalConfig: exp.originalConfig,
    parentArchived: exp.parentArchived ?? false,
    projectName: exp.projectName ?? '',
    projectOwnerId: exp.projectOwnerId ?? 0,
    workspaceId: exp.workspaceId ?? 0,
    workspaceName: exp.workspaceName ?? '',
  };
};

export const mapSearchExperiment = (
  data: Sdk.V1SearchExperimentExperiment,
): types.ExperimentWithTrial => {
  return {
    bestTrial: data.bestTrial && decodeV1TrialToTrialItem(data.bestTrial),
    experiment: data.experiment && mapV1Experiment(data.experiment),
  };
};

export const mapV1Experiment = (
  data: Sdk.V1Experiment,
  jobSummary?: types.JobSummary,
): types.ExperimentItem => {
  const ioConfig = ioTypes.decode<ioTypes.ioTypeExperimentConfig>(
    ioTypes.ioExperimentConfig,
    data.config,
  );
  const continueFn = (value: unknown) => !(value as types.HyperparameterBase).type;
  const hyperparameters = flattenObject<types.HyperparameterBase>(ioConfig.hyperparameters, {
    continueFn,
  }) as types.HyperparametersFlattened;
  return {
    archived: data.archived,
    checkpointCount: data.checkpointCount,
    checkpointSize: parseInt(data?.checkpointSize || '0'),
    config: ioToExperimentConfig(ioConfig),
    configRaw: data.config,
    description: data.description,
    endTime: data.endTime as unknown as string,
    forkedFrom: data.forkedFrom,
    hyperparameters,
    id: data.id,
    jobId: data.jobId,
    jobSummary: jobSummary,
    labels: data.labels || [],
    name: data.name,
    notes: data.notes,
    numTrials: data.numTrials || 0,
    progress: data.progress != null ? data.progress : undefined,
    projectId: data.projectId,
    projectName: data.projectName,
    resourcePool: data.resourcePool || '',
    searcherMetricValue: data.bestTrialSearcherMetric,
    searcherType: data.searcherType,
    startTime: data.startTime as unknown as string,
    state: decodeExperimentState(data.state),
    trialIds: data.trialIds || [],
    userId: data.userId ?? 0,
    workspaceId: data.workspaceId,
    workspaceName: data.workspaceName,
  };
};

export const mapV1ExperimentList = (data: Sdk.V1Experiment[]): types.ExperimentItem[] => {
  // empty JobSummary
  return data.map((e) => mapV1Experiment(e));
};

const filterNonScalarMetrics = (metrics: RawJson): RawJson | undefined => {
  if (!isObject(metrics)) return undefined;
  if (metrics.avgMetrics) {
    return filterNonScalarMetrics(metrics.avgMetrics);
  }
  const scalarMetrics: RawJson = {};
  for (const key in metrics) {
    if (['Infinity', '-Infinity', 'NaN'].includes(metrics[key])) {
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
  Object.entries(data.resources || {}).forEach(([res, val]) => {
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

export const decodeMetrics = (data: Sdk.V1Metrics): types.Metrics => {
  /**
   * using any here because this comes from the api as any
   * however, the protos indicate that it is a Struct/Record
   */
  /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
  const decodeMetricStruct = (data: any): Record<string, number> => {
    const metrics: Record<string, number> = {};
    Object.entries(data || {}).forEach(([metric, value]) => {
      if (typeof metric === 'string' && (typeof value === 'number' || typeof value === 'string')) {
        const numberValue = typeof value === 'number' ? value : parseFloat(value);
        if (!isNaN(numberValue)) metrics[metric] = numberValue;
      }
    });
    return metrics;
  };
  return {
    avgMetrics: decodeMetricStruct(data.avgMetrics),
    batchMetrics: data.batchMetrics?.map(decodeMetricStruct),
  };
};

export const decodeCheckpoint = (data: Sdk.V1Checkpoint): types.CoreApiGenericCheckpoint => {
  const resources: Record<string, number> = {};
  Object.entries(data.resources || {}).forEach(([res, val]) => {
    resources[res] = parseFloat(val);
  });
  return {
    allocationId: data.allocationId,
    experimentConfig: data.training.experimentConfig,
    experimentId: data.training.experimentId,
    hparams: data.training.hparams,
    metadata: data.metadata,
    reportTime: data.reportTime?.toString(),
    resources: resources,
    searcherMetric: data.training.searcherMetric,
    state: decodeCheckpointState(data.state || Sdk.Checkpointv1State.UNSPECIFIED),
    taskId: data.taskId,
    totalBatches: data.metadata['steps_completed'] ?? 0,
    trainingMetrics: data.training.trainingMetrics && decodeMetrics(data.training.trainingMetrics),
    trialId: data.training.trialId,
    uuid: data.uuid,
    validationMetrics:
      data.training.validationMetrics && decodeMetrics(data.training.validationMetrics),
  };
};

export const decodeCheckpoints = (
  data: Sdk.V1GetExperimentCheckpointsResponse,
): types.CheckpointPagination => {
  return {
    checkpoints: data.checkpoints.map(decodeCheckpoint),
    pagination: mapV1Pagination(data.pagination),
  };
};

export const decodeV1TrialToTrialItem = (data: Sdk.Trialv1Trial): types.TrialItem => {
  return {
    autoRestarts: data.restarts,
    bestAvailableCheckpoint: data.bestCheckpoint && decodeCheckpointWorkload(data.bestCheckpoint),
    bestValidationMetric: data.bestValidation && decodeMetricsWorkload(data.bestValidation),
    checkpointCount: data.checkpointCount || 0,
    endTime: data.endTime && (data.endTime as unknown as string),
    experimentId: data.experimentId,
    hyperparameters: flattenObject(data.hparams || {}),
    id: data.id,
    latestValidationMetric: data.latestValidation && decodeMetricsWorkload(data.latestValidation),
    startTime: data.startTime as unknown as string,
    state: decodeExperimentState(data.state),
    totalBatchesProcessed: data.totalBatchesProcessed,
    totalCheckpointSize: parseInt(data?.totalCheckpointSize || '0'),
  };
};

const decodeSummaryMetrics = (data: Sdk.V1DownsampledMetrics[]): types.MetricContainer[] => {
  return data.map((m) => {
    const metrics: types.MetricContainer = {
      data: m.data.map((pt) => ({
        batches: pt.batches,
        epoch: pt.epoch,
        time: pt.time,
        values: pt.values,
      })),
      type:
        m.type === Sdk.V1MetricType.TRAINING
          ? types.MetricType.Training
          : types.MetricType.Validation,
    };
    return metrics;
  });
};

export const decodeTrialSummary = (data: Sdk.V1SummarizeTrialResponse): types.TrialSummary => {
  const trialItem = decodeV1TrialToTrialItem(data.trial);

  return {
    ...trialItem,
    metrics: decodeSummaryMetrics(data.metrics),
  };
};

export const decodeTrialWorkloads = (
  data: Sdk.V1GetTrialWorkloadsResponse,
): types.TrialWorkloads => {
  const workloads = data.workloads.map((ww) => ({
    checkpoint: ww.checkpoint && decodeCheckpointWorkload(ww.checkpoint),
    training: ww.training && decodeMetricsWorkload(ww.training),
    validation: ww.validation && decodeMetricsWorkload(ww.validation),
  }));
  return {
    pagination: data.pagination,
    workloads: workloads,
  };
};

export const decodeTrialResponseToTrialDetails = (
  data: Sdk.V1GetTrialResponse,
): types.TrialDetails => {
  const trialItem = decodeV1TrialToTrialItem(data.trial);
  const EMPTY_STATES = new Set(['UNSPECIFIED', '', undefined]);

  return {
    ...trialItem,
    runnerState: EMPTY_STATES.has(data.trial.runnerState) ? undefined : data.trial.runnerState,
    totalCheckpointSize: Number(data.trial.totalCheckpointSize) || 0,
  };
};

export const jsonToClusterLog = (data: unknown): types.Log => {
  const logData = data as Sdk.V1MasterLogsResponse;
  return {
    id: logData.logEntry?.id ?? 0,
    level: decodeV1LogLevelToLogLevel(logData.logEntry?.level ?? Sdk.V1LogLevel.UNSPECIFIED),
    message: logData.logEntry?.message ?? '',
    time: logData.logEntry?.timestamp as unknown as string,
  };
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

const newlineRegex = /(\r\n|\n)$/g;
const messageRegex = new RegExp(
  [
    '^',
    '\\[([^\\]]+)\\]\\s', // timestamp
    '([0-9a-f]{8})?\\s?', // container id
    '(\\[rank=(\\d+)\\])?\\s?', // rank id
    '(\\|\\|\\s)?', // divider ||
    '((CRITICAL|DEBUG|ERROR|INFO|WARNING):\\s)?', // log level
    '(\\[(\\d+)\\]\\s)?', // process id
    '([\\s\\S]*)', // message
    '$',
  ].join(''),
  'im',
);

const formatLogMessage = (message: string): string => {
  let filteredMessage = message.replace(newlineRegex, '');

  const matches = filteredMessage.match(messageRegex) ?? [];
  if (matches.length === 11) {
    filteredMessage = matches[10] ?? '';

    // process id
    if (matches[9] != null) filteredMessage = `[${matches[9]}] ${filteredMessage}`;

    // rank id
    if (matches[4] != null) filteredMessage = `[rank=${matches[4]}] ${filteredMessage}`;

    // container id
    if (matches[2] != null) filteredMessage = `[${matches[2]}] ${filteredMessage}`;
  }

  return filteredMessage.trim();
};

export const mapV1LogsResponse = <T extends Sdk.V1TrialLogsResponse | Sdk.V1TaskLogsResponse>(
  data: unknown,
): types.TrialLog => {
  const logData = data as T;
  return {
    id: logData.id,
    level: decodeV1LogLevelToLogLevel(logData.level),
    message: formatLogMessage(logData.message),
    time: logData.timestamp as unknown as string,
  };
};

export const mapV1DeviceType = (data: Sdk.Devicev1Type): types.ResourceType => {
  return types.ResourceType[
    data.toString().toUpperCase().replace('TYPE_', '') as keyof typeof types.ResourceType
  ];
};

export const mapV1Workspace = (data: Sdk.V1Workspace): types.Workspace => {
  return {
    ...data,
    pinnedAt: new Date(data.pinnedAt || 0),
    state: mapWorkspaceState(data.state),
  };
};

export const mapDeletionStatus = (
  response: Sdk.V1DeleteProjectResponse | Sdk.V1DeleteWorkspaceResponse,
): types.DeletionStatus => {
  return { completed: response.completed };
};

export const mapWorkspaceState = (state: Sdk.V1WorkspaceState): types.WorkspaceState => {
  return {
    [Sdk.V1WorkspaceState.DELETED]: types.WorkspaceState.Deleted,
    [Sdk.V1WorkspaceState.DELETEFAILED]: types.WorkspaceState.DeleteFailed,
    [Sdk.V1WorkspaceState.DELETING]: types.WorkspaceState.Deleting,
    [Sdk.V1WorkspaceState.UNSPECIFIED]: types.WorkspaceState.Unspecified,
  }[state];
};

export const mapV1Project = (data: Sdk.V1Project): types.Project => {
  return {
    ...data,
    state: mapWorkspaceState(data.state),
    workspaceName: data.workspaceName ?? '',
  };
};

export const mapV1Webhook = (data: Sdk.V1Webhook): types.Webhook => {
  return {
    id: data.id || -1,
    triggers: data.triggers || [],
    url: data.url,
    webhookType:
      {
        [Sdk.V1WebhookType.UNSPECIFIED]: 'Unspecified',
        [Sdk.V1WebhookType.DEFAULT]: 'Default',
        [Sdk.V1WebhookType.SLACK]: 'Slack',
      }[data.webhookType] || 'Unspecified',
  };
};

export const decodeJobStates = (
  states?: Sdk.Jobv1State[],
): Array<
  'STATE_UNSPECIFIED' | 'STATE_QUEUED' | 'STATE_SCHEDULED' | 'STATE_SCHEDULED_BACKFILLED'
> => {
  return states as unknown as Array<
    'STATE_UNSPECIFIED' | 'STATE_QUEUED' | 'STATE_SCHEDULED' | 'STATE_SCHEDULED_BACKFILLED'
  >;
};

export const mapV1ExperimentActionResults = (
  results: Sdk.V1ExperimentActionResult[],
): types.BulkActionResult => {
  return results.reduce(
    (acc, cur) => {
      if (cur.error.length > 0) {
        acc.failed.push(cur);
      } else {
        acc.successful.push(cur.id);
      }
      return acc;
    },
    { failed: [], successful: [] } as types.BulkActionResult,
  );
};

export const decodeProjectColumnsResponse = (r: unknown): ioTypes.ioTypeProjectColumnsResponse =>
  ioTypes.decode(ioTypes.ioProjectColumnsResponse, r);
