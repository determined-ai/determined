import { sha512 } from 'js-sha512';
import queryString from 'query-string';

import { globalStorage } from 'globalStorage';
import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';
import * as decoder from 'services/decoder';
import * as ServiceType from 'services/types';
import * as Type from 'types';

import { noOp } from './utils';

const ApiConfig = new Api.Configuration({
  apiKey: `Bearer ${globalStorage.authToken}`,
  basePath: serverAddress(),
});

export const detApi = {
  Auth: new Api.AuthenticationApi(ApiConfig),
  Cluster: new Api.ClusterApi(ApiConfig),
  Commands: new Api.CommandsApi(ApiConfig),
  Experiments: new Api.ExperimentsApi(ApiConfig),
  Internal: new Api.InternalApi(ApiConfig),
  Notebooks: new Api.NotebooksApi(ApiConfig),
  Shells: new Api.ShellsApi(ApiConfig),
  StreamingCluster: Api.ClusterApiFetchParamCreator(ApiConfig),
  StreamingExperiments: Api.ExperimentsApiFetchParamCreator(ApiConfig),
  StreamingInternal: Api.InternalApiFetchParamCreator(ApiConfig),
  StreamingProfiler: Api.ProfilerApiFetchParamCreator(ApiConfig),
  Templates: new Api.TemplatesApi(ApiConfig),
  TensorBoards: new Api.TensorboardsApi(ApiConfig),
  Users: new Api.UsersApi(ApiConfig),
};

const updatedApiConfigParams = (apiConfig?: Api.ConfigurationParameters):
Api.ConfigurationParameters => {
  return {
    apiKey: `Bearer ${globalStorage.authToken}`,
    basePath: serverAddress(),
    ...apiConfig,
  };
};

// Update references to generated API code with new configuration.
export const updateDetApi = (apiConfig: Api.ConfigurationParameters): void => {
  const config = updatedApiConfigParams(apiConfig);
  detApi.Auth = new Api.AuthenticationApi(config);
  detApi.Cluster = new Api.ClusterApi(config);
  detApi.Commands = new Api.CommandsApi(config);
  detApi.Experiments = new Api.ExperimentsApi(config);
  detApi.Internal = new Api.InternalApi(config);
  detApi.Notebooks = new Api.NotebooksApi(config);
  detApi.Shells = new Api.ShellsApi(config);
  detApi.StreamingCluster = Api.ClusterApiFetchParamCreator(config);
  detApi.StreamingExperiments = Api.ExperimentsApiFetchParamCreator(config);
  detApi.StreamingInternal = Api.InternalApiFetchParamCreator(config);
  detApi.StreamingProfiler = Api.ProfilerApiFetchParamCreator(config);
  detApi.TensorBoards = new Api.TensorboardsApi(config);
  detApi.Users = new Api.UsersApi(config);
  detApi.Templates = new Api.TemplatesApi(config);
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

/* Authentication */

export const login: ServiceType.DetApi<
  Api.V1LoginRequest, Api.V1LoginResponse, ServiceType.LoginResponse
> = {
  name: 'login',
  postProcess: (resp) => ({ token: resp.token, user: decoder.mapV1User(resp.user) }),
  request: (params, options) => detApi.Auth.login(
    { ...params, isHashed: true, password: saltAndHashPassword(params.password) }
    , options,
  ),
};

export const logout: ServiceType.DetApi<
  ServiceType.EmptyParams, Api.V1LogoutResponse, void
> = {
  name: 'logout',
  postProcess: noOp,
  request: () => detApi.Auth.logout(),
};

export const getCurrentUser: ServiceType.DetApi<
  ServiceType.EmptyParams, Api.V1CurrentUserResponse, Type.DetailedUser
> = {
  name: 'getCurrentUser',
  postProcess: (response) => decoder.mapV1User(response.user),
  // We make sure to request using the latest API configuraitonp parameters.
  request: (options) => detApi.Auth.currentUser(options),
};

export const getUsers: ServiceType.DetApi<
  ServiceType.EmptyParams, Api.V1GetUsersResponse, Type.DetailedUser[]
> = {
  name: 'getUsers',
  postProcess: (response) => decoder.mapV1UserList(response),
  request: (options) => detApi.Users.getUsers(options),
};

/* Info */

export const getInfo: ServiceType.DetApi<
  ServiceType.EmptyParams, Api.V1GetMasterResponse, Type.DeterminedInfo
> = {
  name: 'getInfo',
  postProcess: (response) => decoder.mapV1MasterInfo(response),
  request: () => detApi.Cluster.getMaster(),
};

export const getTelemetry: ServiceType.DetApi<
  ServiceType.EmptyParams, Api.V1GetTelemetryResponse, Type.Telemetry
> = {
  name: 'getTelemetry',
  postProcess: (response) => response,
  request: () => detApi.Internal.getTelemetry(),
};

/* Cluster */

export const getAgents: ServiceType.DetApi<
  ServiceType.EmptyParams, Api.V1GetAgentsResponse, Type.Agent[]
> = {
  name: 'getAgents',
  postProcess: (response) => decoder.jsonToAgents(response.agents || []),
  request: () => detApi.Cluster.getAgents(),
};

export const getResourcePools: ServiceType.DetApi<
  ServiceType.EmptyParams, Api.V1GetResourcePoolsResponse, Type.ResourcePool[]
> = {
  name: 'getResourcePools',
  postProcess: (response) => {
    return response.resourcePools?.map(decoder.mapV1ResourcePool) || [];
  },
  request: () => detApi.Internal.getResourcePools(),
};

export const getResourceAllocationAggregated: ServiceType.DetApi<
  ServiceType.GetResourceAllocationAggregatedParams,
  Api.V1ResourceAllocationAggregatedResponse,
  Api.V1ResourceAllocationAggregatedResponse
> = {
  name: 'getResourceAllocationAggregated',
  postProcess: (response) => response,
  request: (params: ServiceType.GetResourceAllocationAggregatedParams, options) => {
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

export const getExperiments: ServiceType.DetApi<
  ServiceType.GetExperimentsParams, Api.V1GetExperimentsResponse, Type.ExperimentPagination
> = {
  name: 'getExperiments',
  postProcess: (response: Api.V1GetExperimentsResponse) => {
    return {
      experiments: decoder.mapV1ExperimentList(response.experiments),
      pagination: response.pagination,
    };
  },
  request: (params: ServiceType.GetExperimentsParams, options) => {
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
      params.users,
      options,
    );
  },
};

export const getExperiment: ServiceType.DetApi<
  ServiceType.GetExperimentParams, Api.V1GetExperimentResponse, Type.ExperimentItem
> = {
  name: 'getExperiment',
  postProcess: (response: Api.V1GetExperimentResponse) =>
    decoder.mapV1Experiment(response.experiment),
  request: (params: ServiceType.GetExperimentParams) => {
    return detApi.Experiments.getExperiment(params.id);
  },
};

export const createExperiment: ServiceType.DetApi<
  ServiceType.CreateExperimentParams, Api.V1CreateExperimentResponse, Type.ExperimentBase
> = {
  name: 'createExperiment',
  postProcess: (resp: Api.V1CreateExperimentResponse) => {
    return decoder.mapV1GetExperimentResponse(resp);
  },
  request: (params: ServiceType.CreateExperimentParams, options) => {
    return detApi.Internal.createExperiment(
      {
        config: params.experimentConfig,
        parentId: params.parentId,
      },
      options,
    );
  },
};

export const archiveExperiment: ServiceType.DetApi<
  ServiceType.ExperimentIdParams, Api.V1ArchiveExperimentResponse, void
> = {
  name: 'archiveExperiment',
  postProcess: noOp,
  request: (params: ServiceType.ExperimentIdParams, options) => {
    return detApi.Experiments.archiveExperiment(params.experimentId, options);
  },
};

export const deleteExperiment: ServiceType.DetApi<
  ServiceType.ExperimentIdParams, Api.V1DeleteExperimentResponse, void
> = {
  name: 'deleteExperiment',
  postProcess: noOp,
  request: (params: ServiceType.ExperimentIdParams, options) => {
    return detApi.Experiments.deleteExperiment(params.experimentId, options);
  },
};

export const unarchiveExperiment: ServiceType.DetApi<
  ServiceType.ExperimentIdParams, Api.V1UnarchiveExperimentResponse, void
> = {
  name: 'unarchiveExperiment',
  postProcess: noOp,
  request: (params: ServiceType.ExperimentIdParams, options) => {
    return detApi.Experiments.unarchiveExperiment(params.experimentId, options);
  },
};

export const activateExperiment: ServiceType.DetApi<
  ServiceType.ExperimentIdParams, Api.V1ActivateExperimentResponse, void
> = {
  name: 'activateExperiment',
  postProcess: noOp,
  request: (params: ServiceType.ExperimentIdParams, options) => {
    return detApi.Experiments.activateExperiment(params.experimentId, options);
  },
};

export const pauseExperiment: ServiceType.DetApi<
  ServiceType.ExperimentIdParams, Api.V1PauseExperimentResponse, void
> = {
  name: 'pauseExperiment',
  postProcess: noOp,
  request: (params: ServiceType.ExperimentIdParams, options) => {
    return detApi.Experiments.pauseExperiment(params.experimentId, options);
  },
};

export const cancelExperiment: ServiceType.DetApi<
  ServiceType.ExperimentIdParams, Api.V1CancelExperimentResponse, void
> = {
  name: 'cancelExperiment',
  postProcess: noOp,
  request: (params: ServiceType.ExperimentIdParams, options) => {
    return detApi.Experiments.cancelExperiment(params.experimentId, options);
  },
};

export const killExperiment: ServiceType.DetApi<
  ServiceType.ExperimentIdParams, Api.V1KillExperimentResponse, void
> = {
  name: 'killExperiment',
  postProcess: noOp,
  request: (params: ServiceType.ExperimentIdParams, options) => {
    return detApi.Experiments.killExperiment(params.experimentId, options);
  },
};

export const patchExperiment: ServiceType.DetApi<
  ServiceType.PatchExperimentParams, Api.V1PatchExperimentResponse, void
> = {
  name: 'patchExperiment',
  postProcess: noOp,
  request: (params: ServiceType.PatchExperimentParams, options) => {
    return detApi.Experiments.patchExperiment(
      params.experimentId,
      params.body as Api.V1Experiment,
      options,
    );
  },
};

export const getExperimentDetails: ServiceType.DetApi<
  ServiceType.ExperimentDetailsParams, Api.V1GetExperimentResponse, Type.ExperimentBase
> = {
  name: 'getExperimentDetails',
  postProcess: (response) => decoder.mapV1GetExperimentResponse(response),
  request: (params, options) => detApi.Experiments.getExperiment(params.id, options),
};

export const getExpValidationHistory: ServiceType.DetApi<
  ServiceType.SingleEntityParams,
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

export const getExpTrials: ServiceType.DetApi<
  ServiceType.GetTrialsParams, Api.V1GetExperimentTrialsResponse, Type.TrialPagination
> = {
  name: 'getExperimentTrials',
  postProcess: (response) => {
    return {
      pagination: response.pagination,
      trials: response.trials.map(trial => decoder.decodeTrialResponseToTrialDetails({ trial })),
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

export const getExperimentLabels: ServiceType.DetApi<
  ServiceType.EmptyParams, Api.V1GetExperimentLabelsResponse, string[]
> = {
  name: 'getExperimentLabels',
  postProcess: (response) => response.labels || [],
  request: (options) => detApi.Experiments.getExperimentLabels(options),
};

export const getTrialDetails: ServiceType.DetApi<
  ServiceType.TrialDetailsParams, Api.V1GetTrialResponse, Type.TrialDetails
> = {
  name: 'getTrialDetails',
  postProcess: (response: Api.V1GetTrialResponse) => {
    return decoder.decodeTrialResponseToTrialDetails(response);
  },
  request: (params: ServiceType.TrialDetailsParams) => detApi.Experiments.getTrial(params.id),
};

/* Tasks */

export const getCommands: ServiceType.DetApi<
  ServiceType.GetCommandsParams, Api.V1GetCommandsResponse, Type.CommandTask[]
> = {
  name: 'getCommands',
  postProcess: (response) => (response.commands || [])
    .map(command => decoder.mapV1Command(command)),
  request: (params: ServiceType.GetCommandsParams) => detApi.Commands.getCommands(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const getJupyterLabs: ServiceType.DetApi<
  ServiceType.GetJupyterLabsParams, Api.V1GetNotebooksResponse, Type.CommandTask[]
> = {
  name: 'getJupyterLabs',
  postProcess: (response) => (response.notebooks || [])
    .map(jupyterLab => decoder.mapV1Notebook(jupyterLab)),
  request: (params: ServiceType.GetJupyterLabsParams) => detApi.Notebooks.getNotebooks(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const getShells: ServiceType.DetApi<
  ServiceType.GetShellsParams, Api.V1GetShellsResponse, Type.CommandTask[]
> = {
  name: 'getShells',
  postProcess: (response) => (response.shells || [])
    .map(shell => decoder.mapV1Shell(shell)),
  request: (params: ServiceType.GetShellsParams) => detApi.Shells.getShells(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const getTensorBoards: ServiceType.DetApi<
  ServiceType.GetTensorBoardsParams, Api.V1GetTensorboardsResponse, Type.CommandTask[]
> = {
  name: 'getTensorBoards',
  postProcess: (response) => (response.tensorboards || [])
    .map(tensorboard => decoder.mapV1TensorBoard(tensorboard)),
  request: (params: ServiceType.GetTensorBoardsParams) => detApi.TensorBoards.getTensorboards(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const killCommand: ServiceType.DetApi<
  ServiceType.CommandIdParams, Api.V1KillCommandResponse, void
> = {
  name: 'killCommand',
  postProcess: noOp,
  request: (params: ServiceType.CommandIdParams) => detApi.Commands
    .killCommand(params.commandId),
};

export const killJupyterLab: ServiceType.DetApi<
  ServiceType.CommandIdParams, Api.V1KillNotebookResponse, void
> = {
  name: 'killJupyterLab',
  postProcess: noOp,
  request: (params: ServiceType.CommandIdParams) => detApi.Notebooks
    .killNotebook(params.commandId),
};

export const killShell: ServiceType.DetApi<
  ServiceType.CommandIdParams, Api.V1KillShellResponse, void
> = {
  name: 'killShell',
  postProcess: noOp,
  request: (params: ServiceType.CommandIdParams) => detApi.Shells
    .killShell(params.commandId),
};

export const killTensorBoard: ServiceType.DetApi<
  ServiceType.CommandIdParams, Api.V1KillTensorboardResponse, void
> = {
  name: 'killTensorBoard',
  postProcess: noOp,
  request: (params: ServiceType.CommandIdParams) => detApi.TensorBoards
    .killTensorboard(params.commandId),
};

export const getTemplates: ServiceType.DetApi<
  ServiceType.GetTemplatesParams, Api.V1GetTemplatesResponse, Type.Template[]
> = {
  name: 'getTemplates',
  postProcess: (response) => (response.templates || [])
    .map(template => decoder.mapV1Template(template)),
  request: (params: ServiceType.GetTemplatesParams) => detApi.Templates.getTemplates(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
    params.name,
  ),
};

export const launchJupyterLab: ServiceType.DetApi<
  ServiceType.LaunchJupyterLabParams, Api.V1LaunchNotebookResponse, Type.CommandTask
> = {
  name: 'launchJupyterLab',
  postProcess: (response) => decoder.mapV1Notebook(response.notebook),
  request: (params: ServiceType.LaunchJupyterLabParams) => detApi.Notebooks
    .launchNotebook(params),
};

export const previewJupyterLab: ServiceType.DetApi<
  ServiceType.LaunchJupyterLabParams, Api.V1LaunchNotebookResponse, Type.RawJson
> = {
  name: 'previewJupyterLab',
  postProcess: (response) => response.config,
  request: (params: ServiceType.LaunchJupyterLabParams) => detApi.Notebooks
    .launchNotebook(params),
};

export const launchTensorBoard: ServiceType.DetApi<
  ServiceType.LaunchTensorBoardParams, Api.V1LaunchTensorboardResponse, Type.CommandTask
> = {
  name: 'launchTensorBoard',
  postProcess: (response) => decoder.mapV1TensorBoard(response.tensorboard),
  request: (params: ServiceType.LaunchTensorBoardParams) => detApi.TensorBoards
    .launchTensorboard(params),
};

/* Logs */

const buildQuery = (params: ServiceType.LogsParams): string => {
  const queryParams: Record<string, number> = {};
  if (params.tail) queryParams['tail'] = params.tail;
  if (params.greaterThanId != null) queryParams['greater_than_id'] = params.greaterThanId;
  return queryString.stringify(queryParams);
};

export const getTaskLogs: ServiceType.HttpApi<ServiceType.TaskLogsParams, Type.Log[]> = {
  httpOptions: (params: ServiceType.TaskLogsParams) => ({
    url: [
      `${commandToEndpoint[params.taskType]}/${params.taskId}/events`,
      buildQuery(params),
    ].join('?'),
  }),
  name: 'getTaskLogs',
  postProcess: response => decoder.jsonToTaskLogs(response.data),
};
