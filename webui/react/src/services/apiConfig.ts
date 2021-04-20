import { sha512 } from 'js-sha512';
import queryString from 'query-string';

import { globalStorage } from 'globalStorage';
import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';
import * as decoder from 'services/decoder';
import {
  CommandIdParams, CreateExperimentParams, DetApi, EmptyParams, ExperimentDetailsParams,
  ExperimentIdParams, GetCommandsParams, GetExperimentsParams, GetNotebooksParams,
  GetResourceAllocationAggregatedParams, GetShellsParams, GetTensorboardsParams, GetTrialsParams,
  HttpApi, LaunchNotebookParams, LaunchTensorboardParams, LoginResponse, LogsParams,
  PatchExperimentParams, SingleEntityParams, TaskLogsParams, TrialDetailsParams,
} from 'services/types';
import {
  Agent, CommandTask, CommandType, DetailedUser, DeterminedInfo, ExperimentBase,
  ExperimentPagination, Log, ResourcePool, Telemetry, TrialDetails, TrialPagination,
  ValidationHistory,
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
  StreamingUnimplemented: Api.UnimplementedApiFetchParamCreator(ApiConfig),
  Tensorboards: new Api.TensorboardsApi(ApiConfig),
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
  detApi.StreamingUnimplemented = Api.UnimplementedApiFetchParamCreator(config);
  detApi.Tensorboards = new Api.TensorboardsApi(config);
  detApi.Users = new Api.UsersApi(config);
};

/* Helpers */

export const saltAndHashPassword = (password?: string): string => {
  if (!password) return '';
  const passwordSalt = 'GubPEmmotfiK9TMD6Zdw';
  return sha512(passwordSalt + password);
};

export const commandToEndpoint: Record<CommandType, string> = {
  [CommandType.Command]: '/commands',
  [CommandType.Notebook]: '/notebooks',
  [CommandType.Tensorboard]: '/tensorboard',
  [CommandType.Shell]: '/shells',
};

/* Authentication */

// export const login: HttpApi<Credentials, LoginResponse> = {
//   httpOptions: ({ password, username }) => {
//     return {
//       body: { password: saltAndHashPassword(password), username },
//       method: 'POST',
//       // task websocket connections still depend on cookies for authentication.
//       url: '/login?cookie=true',
//     };
//   },
//   name: 'login',
//   postProcess: (response) => decoder.jsonToLogin(response.data),
//   unAuthenticated: true,
// };

export const login: DetApi<Api.V1LoginRequest, Api.V1LoginResponse, LoginResponse> = {
  name: 'login',
  postProcess: (resp) => ({ token: resp.token, user: decoder.mapV1User(resp.user) }),
  request: (params) => detApi.Auth.determinedLogin(
    { ...params, isHashed: true, password: saltAndHashPassword(params.password) },
  ),
};

export const logout: DetApi<EmptyParams, Api.V1LogoutResponse, void> = {
  name: 'logout',
  postProcess: noOp,
  request: () => detApi.Auth.determinedLogout(),
};

export const getCurrentUser: DetApi<EmptyParams, Api.V1CurrentUserResponse, DetailedUser> = {
  name: 'getCurrentUser',
  postProcess: (response) => decoder.mapV1User(response.user),
  // We make sure to request using the latest API configuraitonp parameters.
  request: (options) => detApi.Auth.determinedCurrentUser(options),
};

export const getUsers: DetApi<EmptyParams, Api.V1GetUsersResponse, DetailedUser[]> = {
  name: 'getUsers',
  postProcess: (response) => decoder.mapV1UserList(response),
  request: (options) => detApi.Users.determinedGetUsers(options),
};

/* Info */

export const getInfo: DetApi<EmptyParams, Api.V1GetMasterResponse, DeterminedInfo> = {
  name: 'getInfo',
  postProcess: (response) => decoder.jsonToDeterminedInfo(response),
  request: () => detApi.Cluster.determinedGetMaster(),
};

export const getTelemetry: DetApi<EmptyParams, Api.V1GetTelemetryResponse, Telemetry> = {
  name: 'getTelemetry',
  postProcess: (response) => response,
  request: () => detApi.Internal.determinedGetTelemetry(),
};

/* Cluster */

export const getAgents: DetApi<EmptyParams, Api.V1GetAgentsResponse, Agent[]> = {
  name: 'getAgents',
  postProcess: (response) => decoder.jsonToAgents(response.agents || []),
  request: () => detApi.Cluster.determinedGetAgents(),
};

export const getResourcePools: DetApi<EmptyParams, Api.V1GetResourcePoolsResponse, ResourcePool[]> =
{
  name: 'getResourcePools',
  postProcess: (response) => {
    return response.resourcePools?.map(decoder.mapV1ResourcePool) || [];
  },
  request: () => detApi.Internal.determinedGetResourcePools(),
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
    return detApi.Cluster.determinedResourceAllocationAggregated(
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
    return detApi.Experiments.determinedGetExperiments(
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      params.description,
      params.labels,
      params.archived,
      params.states,
      params.users,
      options,
    );
  },
};

export const createExperiment: DetApi<
  CreateExperimentParams, Api.V1CreateExperimentResponse, ExperimentBase
> = {
  name: 'createExperiment',
  postProcess: (resp: Api.V1CreateExperimentResponse) => {
    return decoder.decodeGetV1ExperimentRespToExperimentBase(resp);
  },
  request: (params: CreateExperimentParams, options) => {
    return detApi.Internal.determinedCreateExperiment(
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
    return detApi.Experiments.determinedArchiveExperiment(params.experimentId, options);
  },
};

export const unarchiveExperiment: DetApi<
  ExperimentIdParams, Api.V1UnarchiveExperimentResponse, void
> = {
  name: 'unarchiveExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.determinedUnarchiveExperiment(params.experimentId, options);
  },
};

export const activateExperiment: DetApi<
  ExperimentIdParams, Api.V1ActivateExperimentResponse, void
> = {
  name: 'activateExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.determinedActivateExperiment(params.experimentId, options);
  },
};

export const pauseExperiment: DetApi<ExperimentIdParams, Api.V1PauseExperimentResponse, void> = {
  name: 'pauseExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.determinedPauseExperiment(params.experimentId, options);
  },
};

export const cancelExperiment: DetApi<ExperimentIdParams, Api.V1CancelExperimentResponse, void> = {
  name: 'cancelExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.determinedCancelExperiment(params.experimentId, options);
  },
};

export const killExperiment: DetApi<ExperimentIdParams, Api.V1KillExperimentResponse, void> = {
  name: 'killExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams, options) => {
    return detApi.Experiments.determinedKillExperiment(params.experimentId, options);
  },
};

export const patchExperiment: DetApi<PatchExperimentParams, Api.V1PatchExperimentResponse, void> = {
  name: 'patchExperiment',
  postProcess: noOp,
  request: (params: PatchExperimentParams, options) => {
    return detApi.Experiments.determinedPatchExperiment(
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
  postProcess: (response) => decoder.decodeGetV1ExperimentRespToExperimentBase(response),
  request: (params, options) => detApi.Experiments.determinedGetExperiment(params.id, options),
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
    return detApi.Experiments.determinedGetExperimentValidationHistory(params.id, options);
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
    return detApi.Experiments.determinedGetExperimentTrials(
      params.id,
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
      params.states?.map(state => `STATE_${state.toString()}` as any),
      options,
    );
  },
};

export const getExperimentLabels: DetApi<
  EmptyParams, Api.V1GetExperimentLabelsResponse, string[]
> = {
  name: 'getExperimentLabels',
  postProcess: (response) => response.labels || [],
  request: (options) => detApi.Experiments.determinedGetExperimentLabels(options),
};

export const getTrialDetails: DetApi<
  TrialDetailsParams, Api.V1GetTrialResponse, TrialDetails
> = {
  name: 'getTrialDetails',
  postProcess: (resp: Api.V1GetTrialResponse) => {
    return decoder
      .decodeTrialResponseToTrialDetails(resp);
  },
  request: (params: TrialDetailsParams) => detApi.Experiments.determinedGetTrial(params.id),
};

/* Tasks */

export const getCommands: DetApi<GetCommandsParams, Api.V1GetCommandsResponse, CommandTask[]> = {
  name: 'getCommands',
  postProcess: (response) => (response.commands || [])
    .map(command => decoder.mapV1Command(command)) ,
  request: (params: GetCommandsParams) => detApi.Commands.determinedGetCommands(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const getNotebooks: DetApi<GetNotebooksParams, Api.V1GetNotebooksResponse, CommandTask[]> = {
  name: 'getNotebooks',
  postProcess: (response) => (response.notebooks || [])
    .map(notebook => decoder.mapV1Notebook(notebook)) ,
  request: (params: GetNotebooksParams) => detApi.Notebooks.determinedGetNotebooks(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const getShells: DetApi<GetShellsParams, Api.V1GetShellsResponse, CommandTask[]> = {
  name: 'getShells',
  postProcess: (response) => (response.shells || [])
    .map(shell => decoder.mapV1Shell(shell)) ,
  request: (params: GetShellsParams) => detApi.Shells.determinedGetShells(
    params.sortBy,
    params.orderBy,
    params.offset,
    params.limit,
  ),
};

export const getTensorboards: DetApi<
  GetTensorboardsParams, Api.V1GetTensorboardsResponse, CommandTask[]
> = {
  name: 'getTensorboards',
  postProcess: (response) => (response.tensorboards || [])
    .map(tensorboard => decoder.mapV1Tensorboard(tensorboard)) ,
  request: (params: GetTensorboardsParams) => detApi.Tensorboards.determinedGetTensorboards(
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
    .determinedKillCommand(params.commandId),
};

export const killNotebook: DetApi<CommandIdParams, Api.V1KillNotebookResponse, void> = {
  name: 'killNotebook',
  postProcess: noOp,
  request: (params: CommandIdParams) => detApi.Notebooks
    .determinedKillNotebook(params.commandId),
};

export const killShell: DetApi<CommandIdParams, Api.V1KillShellResponse, void> = {
  name: 'killShell',
  postProcess: noOp,
  request: (params: CommandIdParams) => detApi.Shells
    .determinedKillShell(params.commandId),
};

export const killTensorboard: DetApi<CommandIdParams, Api.V1KillTensorboardResponse, void> = {
  name: 'killTensorboard',
  postProcess: noOp,
  request: (params: CommandIdParams) => detApi.Tensorboards
    .determinedKillTensorboard(params.commandId),
};

export const launchNotebook: DetApi<
  LaunchNotebookParams, Api.V1LaunchNotebookResponse, CommandTask
> = {
  name: 'launchNotebook',
  postProcess: (response) => decoder.mapV1Notebook(response.notebook),
  request: (params: LaunchNotebookParams) => detApi.Notebooks
    .determinedLaunchNotebook(params),
};

export const launchTensorboard: DetApi<
  LaunchTensorboardParams, Api.V1LaunchTensorboardResponse, CommandTask
> = {
  name: 'launchTensorboard',
  postProcess: (response) => decoder.mapV1Tensorboard(response.tensorboard),
  request: (params: LaunchTensorboardParams) => detApi.Tensorboards
    .determinedLaunchTensorboard(params),
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
