import { sha512 } from 'js-sha512';
import queryString from 'query-string';

import { globalStorage } from 'globalStorage';
import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';
import {
  jsonToAgents, jsonToCommands, jsonToDeterminedInfo, jsonToExperimentDetails,
  jsonToLogin, jsonToLogs, jsonToNotebook, jsonToNotebooks, jsonToShells, jsonToTaskLogs,
  jsonToTensorboard, jsonToTensorboards, jsonToTrialDetails, jsonToTrialLogs,jsonToUsers,
} from 'services/decoder';
import * as decoder from 'services/decoder';
import {
  CommandIdParams, CreateExperimentParams, CreateNotebookParams, CreateTensorboardParams, DetApi,
  EmptyParams, ExperimentDetailsParams, ExperimentIdParams,
  GetExperimentsParams, LoginResponse, LogsParams, PatchExperimentParams, TaskLogsParams,
  TrialDetailsParams, TrialLogsParams,
} from 'services/types';
import { HttpApi } from 'services/types';
import {
  Agent, Command, CommandType, Credentials, DetailedUser, DeterminedInfo, ExperimentBase,
  ExperimentDetails, Log, TBSourceType, TrialDetails,
} from 'types';

import { noOp } from './utils';

const ApiConfig = new Api.Configuration({
  apiKey: 'Bearer ' + globalStorage.authToken,
  basePath: serverAddress(),
});

export const detApi = {
  Auth: new Api.AuthenticationApi(ApiConfig),
  Cluster: new Api.ClusterApi(ApiConfig),
  Commands: new Api.CommandsApi(ApiConfig),
  Experiments: new Api.ExperimentsApi(ApiConfig),
  Notebooks: new Api.NotebooksApi(ApiConfig),
  Shells: new Api.ShellsApi(ApiConfig),
  StreamingExperiments: Api.ExperimentsApiFetchParamCreator(ApiConfig),
  StreamingInternal: Api.InternalApiFetchParamCreator(ApiConfig),
  Tensorboards: new Api.TensorboardsApi(ApiConfig),
};

const updatedApiConfigParams = (apiConfig?: Api.ConfigurationParameters):
Api.ConfigurationParameters => {
  return {
    apiKey: 'Bearer ' + globalStorage.authToken,
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
  detApi.Notebooks = new Api.NotebooksApi(config);
  detApi.Shells = new Api.ShellsApi(config);
  detApi.StreamingExperiments = Api.ExperimentsApiFetchParamCreator(config);
  detApi.Tensorboards = new Api.TensorboardsApi(config);
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

export const login: HttpApi<Credentials, LoginResponse> = {
  httpOptions: ({ password, username }) => {
    return {
      body: { password: saltAndHashPassword(password), username },
      method: 'POST',
      // task websocket connections still depend on cookies for authentication.
      url: '/login?cookie=true',
    };
  },
  name: 'login',
  postProcess: (response) => jsonToLogin(response.data),
  unAuthenticated: true,
};

export const logout: DetApi<EmptyParams, Api.V1LogoutResponse, void> = {
  name: 'logout',
  postProcess: noOp,
  request: () => detApi.Auth.determinedLogout(),
};

export const getCurrentUser: DetApi<EmptyParams, Api.V1CurrentUserResponse, DetailedUser> = {
  name: 'getCurrentUser',
  postProcess: (response) => decoder.user(response.user),
  // We make sure to request using the latest API configuraitonp parameters.
  request: (params) => detApi.Auth.determinedCurrentUser(params),
};

export const getUsers: HttpApi<EmptyParams, DetailedUser[]> = {
  httpOptions: () => ({ url: '/users' }),
  name: 'getUsers',
  postProcess: (response) => jsonToUsers(response.data),
};

/* Info */

export const getInfo: HttpApi<EmptyParams, DeterminedInfo> = {
  httpOptions: () => ({ url: '/info' }),
  name: 'getInfo',
  postProcess: (response) => jsonToDeterminedInfo(response.data),
};

/* Agent */

export const getAgents: DetApi<EmptyParams, Api.V1GetAgentsResponse, Agent[]> = {
  name: 'getAgents',
  postProcess: (response) => jsonToAgents(response.agents || []),
  request: () => detApi.Cluster.determinedGetAgents(),
};

/* Experiment */

export const getExperiments: DetApi<
GetExperimentsParams,
Api.V1GetExperimentsResponse,
ExperimentBase[]
> = {
  name: 'activateExperiment',
  postProcess: (response: Api.V1GetExperimentsResponse) => {
    if (response.experiments) {
      return response.experiments.map(
        (experiment: Api.V1Experiment) => experiment as unknown as ExperimentBase,
      );
    }

    return [];
  },
  request: (params: GetExperimentsParams) => detApi.Experiments
    .determinedGetExperiments(
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      params.description,
      params.labels,
      params.archived,
      params.states,
      params.users,
      params.options,
    ),
};

export const createExperiment: DetApi<
CreateExperimentParams, Api.V1CreateExperimentResponse, ExperimentBase> = {
  name: 'createExperiment',
  postProcess: (resp: Api.V1CreateExperimentResponse) => {
    return decoder
      .decodeGetV1ExperimentRespToExperimentBase(resp.experiment, resp.config);
  },
  request: (params: CreateExperimentParams) => detApi.Experiments
    .determinedCreateExperiment({ config: params.experimentConfig, parentId: params.parentId }),
};

export const archiveExperiment: DetApi<
  ExperimentIdParams, Api.V1ArchiveExperimentResponse, void
> = {
  name: 'archiveExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams) => detApi.Experiments
    .determinedArchiveExperiment(params.experimentId),
};

export const unarchiveExperiment: DetApi<
  ExperimentIdParams, Api.V1UnarchiveExperimentResponse, void
> = {
  name: 'unarchiveExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams) => detApi.Experiments
    .determinedUnarchiveExperiment(params.experimentId),
};

export const activateExperiment: DetApi<
  ExperimentIdParams, Api.V1ActivateExperimentResponse, void
> = {
  name: 'activateExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams) => detApi.Experiments
    .determinedActivateExperiment(params.experimentId),
};

export const pauseExperiment: DetApi<ExperimentIdParams, Api.V1PauseExperimentResponse, void> = {
  name: 'pauseExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams) => detApi.Experiments
    .determinedPauseExperiment(params.experimentId),
};

export const cancelExperiment: DetApi<ExperimentIdParams, Api.V1CancelExperimentResponse, void> = {
  name: 'cancelExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams) => detApi.Experiments
    .determinedCancelExperiment(params.experimentId),
};

export const killExperiment: DetApi<ExperimentIdParams, Api.V1KillExperimentResponse, void> = {
  name: 'killExperiment',
  postProcess: noOp,
  request: (params: ExperimentIdParams) => detApi.Experiments
    .determinedKillExperiment(params.experimentId),
};

export const patchExperiment: DetApi<PatchExperimentParams, Api.V1PatchExperimentResponse, void> = {
  name: 'patchExperiment',
  postProcess: noOp,
  request: (params: PatchExperimentParams) => detApi.Experiments
    .determinedPatchExperiment(params.experimentId, params.body as Api.V1Experiment),
};

export const getExperimentDetails: HttpApi<ExperimentDetailsParams, ExperimentDetails> = {
  httpOptions: (params) => ({ url: `/experiments/${params.id}/summary` }),
  name: 'getExperimentDetails',
  postProcess: (response) => jsonToExperimentDetails(response.data),
};

export const getExperimentLabels: DetApi<
  EmptyParams, Api.V1GetExperimentLabelsResponse, string[]
> = {
  name: 'getExperimentLabels',
  postProcess: (response) => response.labels || [],
  request: (params) => detApi.Experiments.determinedGetExperimentLabels(params),
};

export const getTrialDetails: HttpApi<TrialDetailsParams, TrialDetails> = {
  httpOptions: (params: TrialDetailsParams) => ({ url: `/trials/${params.id}/details` }),
  name: 'getTrialDetails',
  postProcess: response => jsonToTrialDetails(response.data),
};

/* Tasks */

export const getCommands: HttpApi<EmptyParams, Command[]> = {
  httpOptions: () => ({ url: '/commands' }),
  name: 'getCommands',
  postProcess: (response) => jsonToCommands(response.data),
};

export const getNotebooks: HttpApi<EmptyParams, Command[]> = {
  httpOptions: () => ({ url: '/notebooks' }),
  name: 'getNotebooks',
  postProcess: (response) => jsonToNotebooks(response.data),
};

export const getShells: HttpApi<EmptyParams, Command[]> = {
  httpOptions: () => ({ url: '/shells' }),
  name: 'getShells',
  postProcess: (response) => jsonToShells(response.data),
};

export const getTensorboards: HttpApi<EmptyParams, Command[]> = {
  httpOptions: () => ({ url: '/tensorboard' }),
  name: 'getTensorboards',
  postProcess: (response) => jsonToTensorboards(response.data),
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

export const createNotebook: HttpApi<CreateNotebookParams, Command> = {
  httpOptions: (params) => {
    return {
      body: {
        config: { resources: { slots: params.slots } },
        context: null,
      },
      method: 'POST',
      url: `${commandToEndpoint[CommandType.Notebook]}`,
    };
  },
  name: 'createNotebook',
  postProcess: (response) => jsonToNotebook(response.data),
};

export const createTensorboard: HttpApi<CreateTensorboardParams, Command> = {
  httpOptions: (params) => {
    const attrName = params.type === TBSourceType.Trial ? 'trial_ids' : 'experiment_ids';
    return {
      body: { [attrName]: params.ids },
      method: 'POST',
      url: `${commandToEndpoint[CommandType.Tensorboard]}`,
    };
  },
  name: 'createTensorboard',
  postProcess: (response) => jsonToTensorboard(response.data),
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
  postProcess: response => jsonToLogs(response.data),
};

export const getTaskLogs: HttpApi<TaskLogsParams, Log[]> = {
  httpOptions: (params: TaskLogsParams) => ({
    url: [
      `${commandToEndpoint[params.taskType]}/${params.taskId}/events`,
      buildQuery(params),
    ].join('?'),
  }),
  name: 'getTaskLogs',
  postProcess: response => jsonToTaskLogs(response.data),
};

export const getTrialLogs: HttpApi<TrialLogsParams, Log[]> = {
  httpOptions: (params: TrialLogsParams) => ({
    url: [
      `/experiments/${params.experimentId}/trials/${params.trialId}/logs`,
      buildQuery(params),
    ].join('?'),
  }),
  name: 'getTrialLogs',
  postProcess: response => jsonToTrialLogs(response.data),
};
