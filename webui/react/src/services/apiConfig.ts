import { sha512 } from 'js-sha512';

import { globalStorage } from 'globalStorage';
import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';
import * as decoder from 'services/decoder';
import * as Service from 'services/types';
import * as Type from 'types';

import { DetApi, EmptyParams, RawJson, SingleEntityParams } from '../shared/types';
import { identity, noOp } from '../shared/utils/service';

const updatedApiConfigParams = (
  apiConfig?: Api.ConfigurationParameters,
): Api.ConfigurationParameters => {
  return {
    apiKey: `Bearer ${globalStorage.authToken}`,
    basePath: serverAddress(),
    ...apiConfig,
  };
};

const generateApiConfig = (apiConfig?: Api.ConfigurationParameters) => {
  const config = updatedApiConfigParams(apiConfig);
  return {
    Auth: new Api.AuthenticationApi(config),
    Cluster: new Api.ClusterApi(config),
    Commands: new Api.CommandsApi(config),
    Experiments: new Api.ExperimentsApi(config),
    Internal: new Api.InternalApi(config),
    Models: new Api.ModelsApi(config),
    Notebooks: new Api.NotebooksApi(config),
    Shells: new Api.ShellsApi(config),
    StreamingCluster: Api.ClusterApiFetchParamCreator(config),
    StreamingExperiments: Api.ExperimentsApiFetchParamCreator(config),
    StreamingInternal: Api.InternalApiFetchParamCreator(config),
    StreamingJobs: Api.JobsApiFetchParamCreator(config),
    StreamingProfiler: Api.ProfilerApiFetchParamCreator(config),
    Tasks: new Api.TasksApi(config),
    Templates: new Api.TemplatesApi(config),
    TensorBoards: new Api.TensorboardsApi(config),
    Users: new Api.UsersApi(config),
  };
};

export let detApi = generateApiConfig();

// Update references to generated API code with new configuration.
export const updateDetApi = (apiConfig: Api.ConfigurationParameters): void => {
  detApi = generateApiConfig(apiConfig);
};

/* Helpers */

export const saltAndHashPassword = (password?: string): string => {
  if (!password) return '';
  const passwordSalt = 'GubPEmmotfiK9TMD6Zdw';
  return sha512(passwordSalt + password);
};

export const commandToEndpoint: Record<Type.CommandType, string> = {
  [Type.CommandType.Command]: '/commands',
  [Type.CommandType.JupyterLab]: '/notebooks',
  [Type.CommandType.TensorBoard]: '/tensorboard',
  [Type.CommandType.Shell]: '/shells',
};

export const getUserIds = (users: string[] = []): number[] | undefined => {
  const userIds = users
    .map(user => parseInt(user))
    .filter(user => !isNaN(user));
  return userIds.length !== 0 ? userIds : undefined;
};

/* Authentication */

export const login: DetApi<
  Api.V1LoginRequest, Api.V1LoginResponse, Service.LoginResponse
> = {
  name: 'login',
  postProcess: (resp) => ({ token: resp.token, user: decoder.mapV1User(resp.user) }),
  request: (params, options) => detApi.Auth.login(
    { ...params, isHashed: true, password: saltAndHashPassword(params.password) }
    , options,
  ),
};

export const logout: DetApi<
  EmptyParams, Api.V1LogoutResponse, void
> = {
  name: 'logout',
  postProcess: noOp,
  request: () => detApi.Auth.logout(),
};

export const getCurrentUser: DetApi<
  EmptyParams, Api.V1CurrentUserResponse, Type.DetailedUser
> = {
  name: 'getCurrentUser',
  postProcess: (response) => decoder.mapV1User(response.user),
  // We make sure to request using the latest API configuraitonp parameters.
  request: (options) => detApi.Auth.currentUser(options),
};

export const getUsers: DetApi<
  EmptyParams, Api.V1GetUsersResponse, Type.DetailedUser[]
> = {
  name: 'getUsers',
  postProcess: (response) => decoder.mapV1UserList(response),
  request: (options) => detApi.Users.getUsers(options),
};

export const setUserPassword: DetApi<
  Service.SetUserPasswordParams, Api.V1SetUserPasswordResponse, Api.V1SetUserPasswordResponse
> = {
  name: 'setUserPassword',
  postProcess: (response) => response,
  request: (params) => detApi.Users.setUserPassword(
    params.userId,
    params.password,
  ),
};

export const patchUser: DetApi<
  Service.PatchUserParams, Api.V1PatchUserResponse, Type.DetailedUser
> = {
  name: 'patchUser',
  postProcess: (response) => decoder.mapV1User(response.user),
  request: (params) => detApi.Users.patchUser(
    params.userId,
    params.userParams,
  ),
};

/* Info */

export const getInfo: DetApi<
  EmptyParams, Api.V1GetMasterResponse, Type.DeterminedInfo
> = {
  name: 'getInfo',
  postProcess: (response) => decoder.mapV1MasterInfo(response),
  request: () => detApi.Cluster.getMaster(),
};

export const getTelemetry: DetApi<
  EmptyParams, Api.V1GetTelemetryResponse, Type.Telemetry
> = {
  name: 'getTelemetry',
  postProcess: (response) => response,
  request: () => detApi.Internal.getTelemetry(),
};

/* Cluster */

export const getAgents: DetApi<
  EmptyParams, Api.V1GetAgentsResponse, Type.Agent[]
> = {
  name: 'getAgents',
  postProcess: (response) => decoder.jsonToAgents(response.agents || []),
  request: () => detApi.Cluster.getAgents(),
};

export const getResourcePools: DetApi<
  EmptyParams, Api.V1GetResourcePoolsResponse, Type.ResourcePool[]
> = {
  name: 'getResourcePools',
  postProcess: (response) => {
    return response.resourcePools?.map(decoder.mapV1ResourcePool) || [];
  },
  request: () => detApi.Internal.getResourcePools(),
};

export const getResourceAllocationAggregated: DetApi<
  Service.GetResourceAllocationAggregatedParams,
  Api.V1ResourceAllocationAggregatedResponse,
  Api.V1ResourceAllocationAggregatedResponse
> = {
  name: 'getResourceAllocationAggregated',
  postProcess: (response) => response,
  request: (params: Service.GetResourceAllocationAggregatedParams, options) => {
    const dateFormat = (params.period === 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY'
      ? 'YYYY-MM' : 'YYYY-MM-DD');
    return detApi.Cluster.resourceAllocationAggregated(
      params.startDate.format(dateFormat),
      params.endDate.format(dateFormat),
      params.period,
      options,
    );
  },
};

/* Experiment */

export const getExperiments: DetApi<
  Service.GetExperimentsParams, Api.V1GetExperimentsResponse, Type.ExperimentPagination
> = {
  name: 'getExperiments',
  postProcess: (response: Api.V1GetExperimentsResponse) => {
    return {
      experiments: decoder.mapV1ExperimentList(response.experiments),
      pagination: response.pagination,
    };
  },
  request: (params: Service.GetExperimentsParams, options) => {
    return detApi.Experiments.getExperiments(
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      params.description,
      params.name,
      params.labels,
      params.archived,
      params.states,
      undefined,
      getUserIds(params.users),
      options,
    );
  },
};

export const getExperiment: DetApi<
  Service.GetExperimentParams, Api.V1GetExperimentResponse, Type.ExperimentItem
> = {
  name: 'getExperiment',
  postProcess: (response: Api.V1GetExperimentResponse) => {
    const exp = decoder.mapV1Experiment(response.experiment);
    exp.jobSummary = response.jobSummary;
    return exp;
  },
  request: (params: Service.GetExperimentParams) => {
    return detApi.Experiments.getExperiment(params.id);
  },
};

export const createExperiment: DetApi<
  Service.CreateExperimentParams, Api.V1CreateExperimentResponse, Type.ExperimentBase
> = {
  name: 'createExperiment',
  postProcess: (resp: Api.V1CreateExperimentResponse) => {
    return decoder.mapV1GetExperimentResponse(resp);
  },
  request: (params: Service.CreateExperimentParams, options) => {
    return detApi.Internal.createExperiment(
      {
        activate: params.activate,
        config: params.experimentConfig,
        parentId: params.parentId,
      },
      options,
    );
  },
};

export const archiveExperiment: DetApi<
  Service.ExperimentIdParams, Api.V1ArchiveExperimentResponse, void
> = {
  name: 'archiveExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.archiveExperiment(params.experimentId, options);
  },
};

export const deleteExperiment: DetApi<
  Service.ExperimentIdParams, Api.V1DeleteExperimentResponse, void
> = {
  name: 'deleteExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.deleteExperiment(params.experimentId, options);
  },
};

export const unarchiveExperiment: DetApi<
  Service.ExperimentIdParams, Api.V1UnarchiveExperimentResponse, void
> = {
  name: 'unarchiveExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.unarchiveExperiment(params.experimentId, options);
  },
};

export const activateExperiment: DetApi<
  Service.ExperimentIdParams, Api.V1ActivateExperimentResponse, void
> = {
  name: 'activateExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.activateExperiment(params.experimentId, options);
  },
};

export const pauseExperiment: DetApi<
  Service.ExperimentIdParams, Api.V1PauseExperimentResponse, void
> = {
  name: 'pauseExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.pauseExperiment(params.experimentId, options);
  },
};

export const cancelExperiment: DetApi<
  Service.ExperimentIdParams, Api.V1CancelExperimentResponse, void
> = {
  name: 'cancelExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.cancelExperiment(params.experimentId, options);
  },
};

export const killExperiment: DetApi<
  Service.ExperimentIdParams, Api.V1KillExperimentResponse, void
> = {
  name: 'killExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.killExperiment(params.experimentId, options);
  },
};

export const patchExperiment: DetApi<
  Service.PatchExperimentParams, Api.V1PatchExperimentResponse, void
> = {
  name: 'patchExperiment',
  postProcess: noOp,
  request: (params: Service.PatchExperimentParams, options) => {
    return detApi.Experiments.patchExperiment(
      params.experimentId,
      params.body as Api.V1Experiment,
      options,
    );
  },
};

export const getExperimentDetails: DetApi<
  Service.ExperimentDetailsParams, Api.V1GetExperimentResponse, Type.ExperimentBase
> = {
  name: 'getExperimentDetails',
  postProcess: (response) => decoder.mapV1GetExperimentResponse(response),
  request: (params, options) => detApi.Experiments.getExperiment(params.id, options),
};

export const getExpValidationHistory: DetApi<
  SingleEntityParams,
  Api.V1GetExperimentValidationHistoryResponse,
  Type.ValidationHistory[]
> = {
  name: 'getExperimentValidationHistory',
  postProcess: (response) => {
    if (!response.validationHistory) return [];
    return response.validationHistory?.map(vh => ({
      endTime: vh.endTime as unknown as string,
      trialId: vh.trialId,
      validationError: vh.searcherMetric,
    }));
  },
  request: (params, options) => {
    return detApi.Experiments.getExperimentValidationHistory(params.id, options);
  },
};

export const getExpTrials: DetApi<
  Service.GetTrialsParams, Api.V1GetExperimentTrialsResponse, Type.TrialPagination
> = {
  name: 'getExperimentTrials',
  postProcess: (response) => {
    return {
      pagination: response.pagination,
      trials: response.trials.map(trial => ({
        workloads: [],
        ...decoder.decodeV1TrialToTrialItem(trial),
      })),
    };
  },
  request: (params, options) => {
    return detApi.Experiments.getExperimentTrials(
      params.id,
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      params.states,
      options,
    );
  },
};

export const getExperimentLabels: DetApi<
  EmptyParams, Api.V1GetExperimentLabelsResponse, string[]
> = {
  name: 'getExperimentLabels',
  postProcess: (response) => response.labels || [],
  request: (options) => detApi.Experiments.getExperimentLabels(options),
};

export const getTrialDetails: DetApi<
  Service.TrialDetailsParams, Api.V1GetTrialResponse, Type.TrialDetails
> = {
  name: 'getTrialDetails',
  postProcess: (response: Api.V1GetTrialResponse) => {
    return decoder.decodeTrialResponseToTrialDetails(response);
  },
  request: (params: Service.TrialDetailsParams) => detApi.Experiments.getTrial(params.id),
};

/* Tasks */

export const getTask: DetApi<
  Service.GetTaskParams, Api.V1GetTaskResponse, Type.TaskItem | undefined
> = {
  name: 'getTask',
  postProcess: (response) => {
    return response.task ? decoder.mapV1Task(response.task) : undefined;
  },
  request: (params: Service.GetTaskParams) => detApi.Tasks.getTask(
    params.taskId,
  ),
};

/* Models */

export const getModels: DetApi<
  Service.GetModelsParams, Api.V1GetModelsResponse, Type.ModelPagination
> = {
  name: 'getModels',
  postProcess: (response) => {
    return {
      models: (response.models).map(model => decoder.mapV1Model(model)),
      pagination: response.pagination,
    };
  },
  request: (params: Service.GetModelsParams) => detApi.Models.getModels(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
    params.name,
    params.description,
    params.labels,
    params.archived,
    undefined,
    getUserIds(params.users),
  ),
};

export const getModel: DetApi<
  Service.GetModelParams, Api.V1GetModelResponse, Type.ModelItem | undefined
> = {
  name: 'getModel',
  postProcess: (response) => {
    return response.model ? decoder.mapV1Model(response.model) : undefined;
  },
  request: (params: Service.GetModelParams) => detApi.Models.getModel(
    params.modelName,
  ),
};

export const getModelDetails: DetApi<
  Service.GetModelDetailsParams, Api.V1GetModelVersionsResponse, Type.ModelVersions | undefined
> = {
  name: 'getModelDetails',
  postProcess: (response) => {
    return (response.model != null && response.modelVersions != null) ?
      decoder.mapV1ModelDetails(response) : undefined;
  },
  request: (params: Service.GetModelDetailsParams) => detApi.Models.getModelVersions(
    params.modelName,
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const getModelVersion: DetApi<
  Service.GetModelVersionParams, Api.V1GetModelVersionResponse, Type.ModelVersion | undefined
> = {
  name: 'getModelVersion',
  postProcess: (response) => {
    return response.modelVersion ? decoder.mapV1ModelVersion(response.modelVersion) : undefined;
  },
  request: (params: Service.GetModelVersionParams) => detApi.Models.getModelVersion(
    params.modelName,
    params.versionId,
  ),
};

export const patchModel: DetApi<
  Service.PatchModelParams, Api.V1PatchModelResponse, Type.ModelItem | undefined>
 = {
   name: 'patchModel',
   postProcess: (response) => response.model ? decoder.mapV1Model(response.model) : undefined,
   request: (params: Service.PatchModelParams) =>
     detApi.Models.patchModel(
       params.modelName,
       params.body,
     ),
 };

export const patchModelVersion: DetApi<
  Service.PatchModelVersionParams, Api.V1PatchModelVersionResponse, Type.ModelVersion | undefined>
 = {
   name: 'patchModelVersion',
   postProcess: (response) => response.modelVersion ?
     decoder.mapV1ModelVersion(response.modelVersion) : undefined,
   request: (params: Service.PatchModelVersionParams) =>
     detApi.Models.patchModelVersion(
       params.modelName,
       params.versionId,
       params.body,
     ),
 };

export const archiveModel: DetApi<
  Service.ArchiveModelParams, Api.V1ArchiveModelResponse, void
> = {
  name: 'archiveModel',
  postProcess: noOp,
  request: (params: Service.GetModelParams) => detApi.Models.archiveModel(
    params.modelName,
  ),
};

export const unarchiveModel: DetApi<
  Service.ArchiveModelParams, Api.V1UnarchiveModelResponse, void
> = {
  name: 'unarchiveModel',
  postProcess: noOp,
  request: (params: Service.GetModelParams) => detApi.Models.unarchiveModel(
    params.modelName,
  ),
};

export const deleteModel: DetApi<
  Service.DeleteModelParams, Api.V1DeleteModelResponse, void
> = {
  name: 'deleteModel',
  postProcess: noOp,
  request: (params: Service.GetModelParams) => detApi.Models.deleteModel(
    params.modelName,
  ),
};

export const deleteModelVersion: DetApi<
  Service.DeleteModelVersionParams, Api.V1DeleteModelVersionResponse, void
> = {
  name: 'deleteModelVersion',
  postProcess: noOp,
  request: (params: Service.GetModelVersionParams) => detApi.Models.deleteModelVersion(
    params.modelName,
    params.versionId,
  ),
};

export const getModelLabels: DetApi<
  EmptyParams, Api.V1GetModelLabelsResponse, string[]
> = {
  name: 'getModelLabels',
  postProcess: (response) => response.labels || [],
  request: (options) => detApi.Models.getModelLabels(options),
};

export const postModel: DetApi<
  Service.PostModelParams, Api.V1PostModelResponse, Type.ModelItem | undefined
> = {
  name: 'postModel',
  postProcess: (response) => {
    return response.model ? decoder.mapV1Model(response.model) : undefined;
  },
  request: (params: Service.PostModelParams) => detApi.Models.postModel({
    description: params.description,
    labels: params.labels,
    metadata: params.metadata,
    name: params.name,
  }),
};

export const postModelVersion: DetApi<
  Service.PostModelVersionParams, Api.V1PostModelVersionResponse, Type.ModelVersion | undefined
> = {
  name: 'postModelVersion',
  postProcess: (response) => {
    return response.modelVersion ? decoder.mapV1ModelVersion(response.modelVersion) : undefined;
  },
  request: (params: Service.PostModelVersionParams) => detApi.Models.postModelVersion(
    params.modelName,
    params.body,
  ),
};

/* Tasks */

const TASK_LIMIT = 1000;

export const getCommands: DetApi<
  Service.GetCommandsParams, Api.V1GetCommandsResponse, Type.CommandTask[]
> = {
  name: 'getCommands',
  postProcess: (response) => (response.commands || [])
    .map(command => decoder.mapV1Command(command)),
  request: (params: Service.GetCommandsParams) => detApi.Commands.getCommands(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit ?? TASK_LIMIT,
    undefined,
    getUserIds(params.users),
  ),
};

export const getJupyterLabs: DetApi<
  Service.GetJupyterLabsParams, Api.V1GetNotebooksResponse, Type.CommandTask[]
> = {
  name: 'getJupyterLabs',
  postProcess: (response) => (response.notebooks || [])
    .map(jupyterLab => decoder.mapV1Notebook(jupyterLab)),
  request: (params: Service.GetJupyterLabsParams) => detApi.Notebooks.getNotebooks(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit ?? TASK_LIMIT,
    undefined,
    getUserIds(params.users),
  ),
};

export const getShells: DetApi<
  Service.GetShellsParams, Api.V1GetShellsResponse, Type.CommandTask[]
> = {
  name: 'getShells',
  postProcess: (response) => (response.shells || [])
    .map(shell => decoder.mapV1Shell(shell)),
  request: (params: Service.GetShellsParams) => detApi.Shells.getShells(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit ?? TASK_LIMIT,
    undefined,
    getUserIds(params.users),
  ),
};

export const getTensorBoards: DetApi<
  Service.GetTensorBoardsParams, Api.V1GetTensorboardsResponse, Type.CommandTask[]
> = {
  name: 'getTensorBoards',
  postProcess: (response) => (response.tensorboards || [])
    .map(tensorboard => decoder.mapV1TensorBoard(tensorboard)),
  request: (params: Service.GetTensorBoardsParams) => detApi.TensorBoards.getTensorboards(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit ?? TASK_LIMIT,
    undefined,
    getUserIds(params.users),
  ),
};

export const killCommand: DetApi<
  Service.CommandIdParams, Api.V1KillCommandResponse, void
> = {
  name: 'killCommand',
  postProcess: noOp,
  request: (params: Service.CommandIdParams) => detApi.Commands
    .killCommand(params.commandId),
};

export const killJupyterLab: DetApi<
  Service.CommandIdParams, Api.V1KillNotebookResponse, void
> = {
  name: 'killJupyterLab',
  postProcess: noOp,
  request: (params: Service.CommandIdParams) => detApi.Notebooks
    .killNotebook(params.commandId),
};

export const killShell: DetApi<
  Service.CommandIdParams, Api.V1KillShellResponse, void
> = {
  name: 'killShell',
  postProcess: noOp,
  request: (params: Service.CommandIdParams) => detApi.Shells
    .killShell(params.commandId),
};

export const killTensorBoard: DetApi<
  Service.CommandIdParams, Api.V1KillTensorboardResponse, void
> = {
  name: 'killTensorBoard',
  postProcess: noOp,
  request: (params: Service.CommandIdParams) => detApi.TensorBoards
    .killTensorboard(params.commandId),
};

export const getTemplates: DetApi<
  Service.GetTemplatesParams, Api.V1GetTemplatesResponse, Type.Template[]
> = {
  name: 'getTemplates',
  postProcess: (response) => (response.templates || [])
    .map(template => decoder.mapV1Template(template)),
  request: (params: Service.GetTemplatesParams) => detApi.Templates.getTemplates(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
    params.name,
  ),
};

export const launchJupyterLab: DetApi<
  Service.LaunchJupyterLabParams, Api.V1LaunchNotebookResponse, Type.CommandTask
> = {
  name: 'launchJupyterLab',
  postProcess: (response) => decoder.mapV1Notebook(response.notebook),
  request: (params: Service.LaunchJupyterLabParams) => detApi.Notebooks
    .launchNotebook(params),
};

export const previewJupyterLab: DetApi<
  Service.LaunchJupyterLabParams, Api.V1LaunchNotebookResponse, RawJson
> = {
  name: 'previewJupyterLab',
  postProcess: (response) => response.config,
  request: (params: Service.LaunchJupyterLabParams) => detApi.Notebooks
    .launchNotebook(params),
};

export const launchTensorBoard: DetApi<
  Service.LaunchTensorBoardParams, Api.V1LaunchTensorboardResponse, Type.CommandTask
> = {
  name: 'launchTensorBoard',
  postProcess: (response) => decoder.mapV1TensorBoard(response.tensorboard),
  request: (params: Service.LaunchTensorBoardParams) => detApi.TensorBoards
    .launchTensorboard(params),
};

/* Jobs */

export const getJobQueue: DetApi<
  Service.GetJobQParams, Api.V1GetJobsResponse, Service.GetJobsResponse
> = {
  name: 'getJobQ',
  postProcess: (response) => {
    response.jobs = response.jobs.filter(job => !!job.summary);
    // we don't work with jobs without a summary in the ui yet
    return response as Service.GetJobsResponse;
  },
  request: (params: Service.GetJobQParams) => detApi.Internal.getJobs(
    params.offset,
    params.limit,
    params.resourcePool,
    params.orderBy,
  ),
};

export const getJobQueueStats: DetApi<
  Service.GetJobQStatsParams,
  Api.V1GetJobQueueStatsResponse,
  Api.V1GetJobQueueStatsResponse
> = {
  name: 'getJobQStats',
  postProcess: identity,
  request: ({ resourcePools }) => detApi.Internal.getJobQueueStats(resourcePools),
};

export const updateJobQueue: DetApi<
  Api.V1UpdateJobQueueRequest,
  Api.V1UpdateJobQueueResponse,
  Api.V1UpdateJobQueueResponse
> = {
  name: 'updateJobQueue',
  postProcess: identity,
  request: (params: Api.V1UpdateJobQueueRequest) => detApi.Internal.updateJobQueue(params),
};
