import { sha512 } from 'js-sha512';
import queryString from 'query-string';

import { decode, ioTypeUser, ioUser } from 'ioTypes';
import { Api } from 'services/apiBuilder';
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
import {
  Agent, Command, CommandType, Credentials, DeterminedInfo, Experiment, ExperimentDetails,
  Log, TBSourceType, TrialDetails, User,
} from 'types';

/* Helpers */

const saltAndHashPassword = (password?: string): string => {
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

export const login: Api<Credentials, void> = {
  httpOptions: ({ password, username }) => {
    return {
      body: { password: saltAndHashPassword(password), username },
      method: 'POST',
      url: '/login?cookie=true',
    };
  },
  name: 'login',
};

export const getCurrentUser: Api<EmptyParams, User> = {
  httpOptions: () => ({ url: '/users/me' }),
  name: 'getCurrentUser',
  postProcess: (response) => {
    const result = decode<ioTypeUser>(ioUser, response.data);
    return {
      id: result.id,
      isActive: result.active,
      isAdmin: result.admin,
      username: result.username,
    };
  },
};

export const getUsers: Api<EmptyParams, User[]> = {
  httpOptions: () => ({ url: '/users' }),
  name: 'getUsers',
  postProcess: (response) => jsonToUsers(response.data),
};

/* Info */

export const getInfo: Api<EmptyParams, DeterminedInfo> = {
  httpOptions: () => ({ url: '/info' }),
  name: 'getInfo',
  postProcess: (response) => jsonToDeterminedInfo(response.data),
};

/* Agent */

export const getAgents: Api<EmptyParams, Agent[]> = {
  httpOptions: () => ({ url: '/agents' }),
  name: 'getAgents',
  postProcess: (response) => jsonToAgents(response.data),
};

/* Experiment */

export const forkExperiment: Api<ForkExperimentParams, number> = {
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

export const patchExperiment: Api<PatchExperimentParams, void> = {
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

export const killExperiment: Api<KillExpParams, void> = {
  httpOptions: (params) => {
    return {
      method: 'POST',
      url: `/experiments/${params.experimentId.toString()}/kill`,
    };
  },
  name: 'killExperiment',
};

export const getExperimentSummaries: Api<ExperimentsParams, Experiment[]> = {
  httpOptions: (params) => ({
    url: '/experiment-summaries' + (params.states ? '?states='+params.states.join(',') : ''),
  }),
  name: 'getExperimentSummaries',
  postProcess: (response) => jsonToExperiments(response.data),
};

export const getExperimentDetails: Api<ExperimentDetailsParams, ExperimentDetails> = {
  httpOptions: (params) => ({
    url: `/experiments/${params.id}/summary`,
  }),
  name: 'getExperimentDetails',
  postProcess: (response) => jsonToExperimentDetails(response.data),
};

export const getTrialDetails: Api<TrialDetailsParams, TrialDetails> = {
  httpOptions: (params: TrialDetailsParams) => ({
    url: `/trials/${params.id}/details`,
  }),
  name: 'getTrialDetails',
  postProcess: response => jsonToTrialDetails(response.data),
};

/* Tasks */

export const getCommands: Api<EmptyParams, Command[]> = {
  httpOptions: () => ({ url: '/commands' }),
  name: 'getCommands',
  postProcess: (response) => jsonToCommands(response.data),
};

export const getNotebooks: Api<EmptyParams, Command[]> = {
  httpOptions: () => ({ url: '/notebooks' }),
  name: 'getNotebooks',
  postProcess: (response) => jsonToNotebooks(response.data),
};

export const getShells: Api<EmptyParams, Command[]> = {
  httpOptions: () => ({ url: '/shells' }),
  name: 'getShells',
  postProcess: (response) => jsonToShells(response.data),
};

export const getTensorboards: Api<EmptyParams, Command[]> = {
  httpOptions: () => ({ url: '/tensorboard' }),
  name: 'getTensorboards',
  postProcess: (response) => jsonToTensorboards(response.data),
};

export const killCommand: Api<KillCommandParams, void> = {
  httpOptions: (params) => {
    return {
      method: 'DELETE',
      url: `${commandToEndpoint[params.commandType]}/${params.commandId}`,
    };
  },
  name: 'killCommand',
};

export const createNotebook: Api<CreateNotebookParams, Command> = {
  httpOptions: (params) => {
    return {
      body: {
        config: {
          resources: { slots: params.slots },
        },
        context: null,
      },
      method: 'POST',
      url: `${commandToEndpoint[CommandType.Notebook]}`,
    };
  },
  name: 'createNotebook',
  postProcess: (response) => jsonToNotebook(response.data),
};

export const createTensorboard: Api<CreateTensorboardParams, Command> = {
  httpOptions: (params) => {
    const attrName = params.type === TBSourceType.Trial ? 'trial_ids' : 'experiment_ids';
    return {
      body: {
        [attrName]: params.ids,
      },
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

export const getMasterLogs: Api<LogsParams, Log[]> = {
  httpOptions: (params: LogsParams) => ({
    url: [ '/logs', buildQuery(params) ].join('?'),
  }),
  name: 'getMasterLogs',
  postProcess: response => jsonToLogs(response.data),
};

export const getTaskLogs: Api<TaskLogsParams, Log[]> = {
  httpOptions: (params: TaskLogsParams) => ({
    url: [
      `${commandToEndpoint[params.taskType]}/${params.taskId}/events`,
      buildQuery(params),
    ].join('?'),
  }),
  name: 'getTaskLogs',
  postProcess: response => jsonToTaskLogs(response.data),
};

export const getTrialLogs: Api<TrialLogsParams, Log[]> = {
  httpOptions: (params: TrialLogsParams) => ({
    url: [ `/trials/${params.trialId}/logs`, buildQuery(params) ].join('?'),
  }),
  name: 'getTrialLogs',
  postProcess: response => jsonToTrialLogs(response.data),
};
