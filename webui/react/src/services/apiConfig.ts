import { sha512 } from 'js-sha512';
import queryString from 'query-string';

import { decode, ioDetailedUser, ioTypeDetailedUser } from 'ioTypes';
import {
  jsonToAgents, jsonToCommands, jsonToDeterminedInfo,
  jsonToExperimentDetails, jsonToExperiments, jsonToLogs, jsonToNotebook, jsonToNotebooks,
  jsonToShells, jsonToTaskLogs, jsonToTensorboard, jsonToTensorboards, jsonToTrialDetails,
  jsonToTrialLogs,jsonToUsers,
} from 'services/decoder';
import {
  CreateNotebookParams, CreateTensorboardParams,
  EmptyParams, ExperimentDetailsParams, ExperimentsParams,
  ForkExperimentParams, KillCommandParams, KillExpParams, LogsParams, PatchExperimentParams,
  TaskLogsParams, TrialDetailsParams, TrialLogsParams,
} from 'services/types';
import { HttpApi } from 'services/types';
import {
  Agent, Command, CommandType, Credentials, DetailedUser, DeterminedInfo, ExperimentBase,
  ExperimentDetails, Log, TBSourceType, TrialDetails,
} from 'types';

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

export const login: HttpApi<Credentials, void> = {
  httpOptions: ({ password, username }) => {
    return {
      body: { password: saltAndHashPassword(password), username },
      method: 'POST',
      url: '/login?cookie=true',
    };
  },
  name: 'login',
};

export const getCurrentUser: HttpApi<EmptyParams, DetailedUser> = {
  httpOptions: () => ({ url: '/users/me' }),
  name: 'getCurrentUser',
  postProcess: (response) => {
    const result = decode<ioTypeDetailedUser>(ioDetailedUser, response.data);
    return {
      id: result.id,
      isActive: result.active,
      isAdmin: result.admin,
      username: result.username,
    };
  },
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

export const getAgents: HttpApi<EmptyParams, Agent[]> = {
  httpOptions: () => ({ url: '/agents' }),
  name: 'getAgents',
  postProcess: (response) => jsonToAgents(response.data),
};

/* Experiment */

export const forkExperiment: HttpApi<ForkExperimentParams, number> = {
  httpOptions: (params) => {
    return {
      body: {
        experiment_config: params.experimentConfig,
        parent_id: params.parentId,
      },
      headers: { 'content-type': 'application/json', 'withCredentials': true },
      method: 'POST',
      url: '/experiments',
    };
  },
  name: 'forkExperiment',
  /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
  postProcess: (response: any) => response.data.id,
};

export const patchExperiment: HttpApi<PatchExperimentParams, void> = {
  httpOptions: (params) => {
    return {
      body: params.body,
      headers: { 'content-type': 'application/merge-patch+json', 'withCredentials': true },
      method: 'PATCH',
      url: `/experiments/${params.experimentId.toString()}`,
    };
  },
  name: 'patchExperiment',
};

export const killExperiment: HttpApi<KillExpParams, void> = {
  httpOptions: (params) => {
    return {
      method: 'POST',
      url: `/experiments/${params.experimentId.toString()}/kill`,
    };
  },
  name: 'killExperiment',
};

export const getExperimentSummaries: HttpApi<ExperimentsParams, ExperimentBase[]> = {
  httpOptions: (params) => ({
    url: [
      '/experiment-summaries',
      params.states ? `?states=${params.states.join(',')}` : '',
    ].join(''),
  }),
  name: 'getExperimentSummaries',
  postProcess: (response) => jsonToExperiments(response.data),
};

export const getExperimentDetails: HttpApi<ExperimentDetailsParams, ExperimentDetails> = {
  httpOptions: (params) => ({ url: `/experiments/${params.id}/summary` }),
  name: 'getExperimentDetails',
  postProcess: (response) => jsonToExperimentDetails(response.data),
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

export const killCommand: HttpApi<KillCommandParams, void> = {
  httpOptions: (params) => {
    return {
      method: 'DELETE',
      url: `${commandToEndpoint[params.commandType]}/${params.commandId}`,
    };
  },
  name: 'killCommand',
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
      `/trials/${params.trialId}/logs`,
      buildQuery(params),
    ].join('?'),
  }),
  name: 'getTrialLogs',
  postProcess: response => jsonToTrialLogs(response.data),
};
