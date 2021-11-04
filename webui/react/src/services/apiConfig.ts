import { sha512 } from 'js-sha512';
import queryString from 'query-string';

import { globalStorage } from 'globalStorage';
import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';
import * as decoder from 'services/decoder';
import {
  CommandIdParams, CreateExperimentParams, DetApi, EmptyParams,
  ExperimentDetailsParams, ExperimentIdParams, GetCommandsParams,
  GetExperimentParams, GetExperimentsParams, GetJupyterLabsParams,
  GetResourceAllocationAggregatedParams, GetShellsParams, GetTemplatesParams, GetTensorBoardsParams,
  GetTrialsParams, HttpApi, LaunchJupyterLabParams, LaunchTensorBoardParams, LoginResponse,
  LogsParams, PatchExperimentParams, SingleEntityParams, TaskLogsParams, TrialDetailsParams,
} from 'services/types';
import {
  Agent, CommandTask, CommandType, DetailedUser, DeterminedInfo, ExperimentBase,
  ExperimentItem,
  ExperimentPagination, Log, RawJson, ResourcePool, Telemetry, Template, TrialDetails,
  TrialPagination, ValidationHistory,
} from 'types';

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

export const commandToEndpoint: Record<CommandType, string> = {
  [CommandType.Command]: '/commands',
  [CommandType.JupyterLab]: '/notebooks',
  [CommandType.TensorBoard]: '/tensorboard',
  [CommandType.Shell]: '/shells',
};

/* Authentication */

export const login: DetApi<Api.V1LoginRequest, Api.V1LoginResponse, LoginResponse> = {
  name: 'login',
  postProcess: (resp) => ({ token: resp.token, user: decoder.mapV1User(resp.user) }),
  request: (params, options) => detApi.Auth.login(
    { ...params, isHashed: true, password: saltAndHashPassword(params.password) }
    , options,
  ),
};

export const logout: DetApi<EmptyParams, Api.V1LogoutResponse, void> = {
  name: 'logout',
  postProcess: noOp,
  request: () => detApi.Auth.logout(),
};

export const getCurrentUser: DetApi<EmptyParams, Api.V1CurrentUserResponse, DetailedUser> = {
  name: 'getCurrentUser',
  postProcess: (response) => decoder.mapV1User(response.user),
  // We make sure to request using the latest API configuraitonp parameters.
  request: (options) => detApi.Auth.currentUser(options),
};

export const getUsers: DetApi<EmptyParams, Api.V1GetUsersResponse, DetailedUser[]> = {
  name: 'getUsers',
  postProcess: (response) => decoder.mapV1UserList(response),
  request: (options) => detApi.Users.getUsers(options),
};

/* Info */

export const getInfo: DetApi<EmptyParams, Api.V1GetMasterResponse, DeterminedInfo> = {
  name: 'getInfo',
  postProcess: (response) => decoder.mapV1MasterInfo(response),
  request: () => detApi.Cluster.getMaster(),
};

export const getTelemetry: DetApi<EmptyParams, Api.V1GetTelemetryResponse, Telemetry> = {
  name: 'getTelemetry',
  postProcess: (response) => response,
  request: () => detApi.Internal.getTelemetry(),
};

/* Cluster */

export const getAgents: DetApi<EmptyParams, Api.V1GetAgentsResponse, Agent[]> = {
  name: 'getAgents',
  postProcess: (response) => decoder.jsonToAgents(response.agents || []),
  request: () => detApi.Cluster.getAgents(),
};

export const getResourcePools: DetApi<EmptyParams, Api.V1GetResourcePoolsResponse, ResourcePool[]> =
{
  name: 'getResourcePools',
  postProcess: (response) => {
    return response.resourcePools?.map(decoder.mapV1ResourcePool) || [];
  },
  request: () => detApi.Internal.getResourcePools(),
};

export const getResourceAllocationAggregated: DetApi<
  GetResourceAllocationAggregatedParams, Api.V1ResourceAllocationAggregatedResponse,
  Api.V1ResourceAllocationAggregatedResponse
> = {
  name: 'getResourceAllocationAggregated',
  postProcess: (response) => response,
  request: (params: GetResourceAllocationAggregatedParams, options) => {
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
  GetExperimentsParams, Api.V1GetExperimentsResponse, ExperimentPagination
> = {
  name: 'getExperiments',
  postProcess: (response: Api.V1GetExperimentsResponse) => {
    return {
      experiments: decoder.mapV1ExperimentList(response.experiments),
      pagination: response.pagination,
    };
  },
  request: (params: GetExperimentsParams, options) => {
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

export const getExperiment: DetApi<
GetExperimentParams, Api.V1GetExperimentResponse, ExperimentItem
> = {
  name: 'getExperiment',
  postProcess: (response: Api.V1GetExperimentResponse) =>
    decoder.mapV1Experiment(response.experiment),
  request: (params: GetExperimentParams) => {
    return detApi.Experiments.getExperiment(params.id);
  },
};

export const createExperiment: DetApi<
  CreateExperimentParams, Api.V1CreateExperimentResponse, ExperimentBase
> = {
  name: 'createExperiment',
  postProcess: (resp: Api.V1CreateExperimentResponse) => {
    return decoder.mapV1GetExperimentResponse(resp);
  },
  request: (params: CreateExperimentParams, options) => {
    return detApi.Internal.createExperiment(
      {
        config: params.experimentConfig,
        parentId: params.parentId,
      },
      options,
    );
  },
};

export const archiveExperiment: DetApi<
  ExperimentIdParams, Api.V1ArchiveExperimentResponse, void
> = {
  name: 'archiveExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.archiveExperiment(params.experimentId, options);
  },
};

export const deleteExperiment: DetApi<
  ExperimentIdParams, Api.V1DeleteExperimentResponse, void
> = {
  name: 'deleteExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.deleteExperiment(params.experimentId, options);
  },
};

export const unarchiveExperiment: DetApi<
  ExperimentIdParams, Api.V1UnarchiveExperimentResponse, void
> = {
  name: 'unarchiveExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.unarchiveExperiment(params.experimentId, options);
  },
};

export const activateExperiment: DetApi<
  ExperimentIdParams, Api.V1ActivateExperimentResponse, void
> = {
  name: 'activateExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.activateExperiment(params.experimentId, options);
  },
};

export const pauseExperiment: DetApi<ExperimentIdParams, Api.V1PauseExperimentResponse, void> = {
  name: 'pauseExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.pauseExperiment(params.experimentId, options);
  },
};

export const cancelExperiment: DetApi<ExperimentIdParams, Api.V1CancelExperimentResponse, void> = {
  name: 'cancelExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.cancelExperiment(params.experimentId, options);
  },
};

export const killExperiment: DetApi<ExperimentIdParams, Api.V1KillExperimentResponse, void> = {
  name: 'killExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.killExperiment(params.experimentId, options);
  },
};

export const patchExperiment: DetApi<PatchExperimentParams, Api.V1PatchExperimentResponse, void> = {
  name: 'patchExperiment',
  postProcess: noOp,
  request: (params: PatchExperimentParams, options) => {
    return detApi.Experiments.patchExperiment(
      params.experimentId,
      params.body as Api.V1Experiment,
      options,
    );
  },
};

export const getExperimentDetails: DetApi<
  ExperimentDetailsParams, Api.V1GetExperimentResponse, ExperimentBase
> = {
  name: 'getExperimentDetails',
  postProcess: (response) => decoder.mapV1GetExperimentResponse(response),
  request: (params, options) => detApi.Experiments.getExperiment(params.id, options),
};

export const getExpValidationHistory: DetApi<
  SingleEntityParams, Api.V1GetExperimentValidationHistoryResponse, ValidationHistory[]
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
  GetTrialsParams, Api.V1GetExperimentTrialsResponse, TrialPagination
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

export const getExperimentLabels: DetApi<
  EmptyParams, Api.V1GetExperimentLabelsResponse, string[]
> = {
  name: 'getExperimentLabels',
  postProcess: (response) => response.labels || [],
  request: (options) => detApi.Experiments.getExperimentLabels(options),
};

export const getTrialDetails: DetApi<
  TrialDetailsParams, Api.V1GetTrialResponse, TrialDetails
> = {
  name: 'getTrialDetails',
  postProcess: (resp: Api.V1GetTrialResponse) => {
    return decoder
      .decodeTrialResponseToTrialDetails(resp);
  },
  request: (params: TrialDetailsParams) => detApi.Experiments.getTrial(params.id),
};

/* Tasks */

export const getCommands: DetApi<GetCommandsParams, Api.V1GetCommandsResponse, CommandTask[]> = {
  name: 'getCommands',
  postProcess: (response) => (response.commands || [])
    .map(command => decoder.mapV1Command(command)),
  request: (params: GetCommandsParams) => detApi.Commands.getCommands(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const getJupyterLabs: DetApi<
  GetJupyterLabsParams, Api.V1GetNotebooksResponse, CommandTask[]
> = {
  name: 'getJupyterLabs',
  postProcess: (response) => (response.notebooks || [])
    .map(jupyterLab => decoder.mapV1Notebook(jupyterLab)),
  request: (params: GetJupyterLabsParams) => detApi.Notebooks.getNotebooks(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const getShells: DetApi<GetShellsParams, Api.V1GetShellsResponse, CommandTask[]> = {
  name: 'getShells',
  postProcess: (response) => (response.shells || [])
    .map(shell => decoder.mapV1Shell(shell)),
  request: (params: GetShellsParams) => detApi.Shells.getShells(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const getTensorBoards: DetApi<
  GetTensorBoardsParams, Api.V1GetTensorboardsResponse, CommandTask[]
> = {
  name: 'getTensorBoards',
  postProcess: (response) => (response.tensorboards || [])
    .map(tensorboard => decoder.mapV1TensorBoard(tensorboard)),
  request: (params: GetTensorBoardsParams) => detApi.TensorBoards.getTensorboards(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const killCommand: DetApi<CommandIdParams, Api.V1KillCommandResponse, void> = {
  name: 'killCommand',
  postProcess: noOp,
  request: (params: CommandIdParams) => detApi.Commands
    .killCommand(params.commandId),
};

export const killJupyterLab: DetApi<CommandIdParams, Api.V1KillNotebookResponse, void> = {
  name: 'killJupyterLab',
  postProcess: noOp,
  request: (params: CommandIdParams) => detApi.Notebooks
    .killNotebook(params.commandId),
};

export const killShell: DetApi<CommandIdParams, Api.V1KillShellResponse, void> = {
  name: 'killShell',
  postProcess: noOp,
  request: (params: CommandIdParams) => detApi.Shells
    .killShell(params.commandId),
};

export const killTensorBoard: DetApi<CommandIdParams, Api.V1KillTensorboardResponse, void> = {
  name: 'killTensorBoard',
  postProcess: noOp,
  request: (params: CommandIdParams) => detApi.TensorBoards
    .killTensorboard(params.commandId),
};

export const getTemplates: DetApi<GetTemplatesParams, Api.V1GetTemplatesResponse, Template[]> = {
  name: 'getTemplates',
  postProcess: (response) => (response.templates || [])
    .map(template => decoder.mapV1Template(template)),
  request: (params: GetTemplatesParams) => detApi.Templates.getTemplates(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
    params.name,
  ),
};

export const launchJupyterLab: DetApi<
  LaunchJupyterLabParams, Api.V1LaunchNotebookResponse, CommandTask
> = {
  name: 'launchJupyterLab',
  postProcess: (response) => decoder.mapV1Notebook(response.notebook),
  request: (params: LaunchJupyterLabParams) => detApi.Notebooks
    .launchNotebook(params),
};

export const previewJupyterLab: DetApi<
  LaunchJupyterLabParams, Api.V1LaunchNotebookResponse, RawJson
> = {
  name: 'previewJupyterLab',
  postProcess: (response) => response.config,
  request: (params: LaunchJupyterLabParams) => detApi.Notebooks
    .launchNotebook(params),
};

export const launchTensorBoard: DetApi<
  LaunchTensorBoardParams, Api.V1LaunchTensorboardResponse, CommandTask
> = {
  name: 'launchTensorBoard',
  postProcess: (response) => decoder.mapV1TensorBoard(response.tensorboard),
  request: (params: LaunchTensorBoardParams) => detApi.TensorBoards
    .launchTensorboard(params),
};

/* Logs */

const buildQuery = (params: LogsParams): string => {
  const queryParams: Record<string, number> = {};
  if (params.tail) queryParams['tail'] = params.tail;
  if (params.greaterThanId != null) queryParams['greater_than_id'] = params.greaterThanId;
  return queryString.stringify(queryParams);
};

export const getMasterLogs: HttpApi<LogsParams, Log[]> = {
  httpOptions: (params: LogsParams) => ({ url: [ '/logs', buildQuery(params) ].join('?') }),
  name: 'getMasterLogs',
  postProcess: response => decoder.jsonToLogs(response.data),
};

export const getTaskLogs: HttpApi<TaskLogsParams, Log[]> = {
  httpOptions: (params: TaskLogsParams) => ({
    url: [
      `${commandToEndpoint[params.taskType]}/${params.taskId}/events`,
      buildQuery(params),
    ].join('?'),
  }),
  name: 'getTaskLogs',
  postProcess: response => decoder.jsonToTaskLogs(response.data),
};
