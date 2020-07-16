import { sha512 } from 'js-sha512';
import queryString from 'query-string';

import { decode, ioTypeUser, ioUser } from 'ioTypes';
import { Api } from 'services/apiBuilder';
import {
  jsonToCommandLogs, jsonToDeterminedInfo, jsonToExperimentDetails,
  jsonToExperiments, jsonToLogs, jsonToTensorboard, jsonToTrialDetails, jsonToTrialLogs,
} from 'services/decoder';
import {
  CommandLogsParams, ExperimentDetailsParams, ExperimentsParams, KillCommandParams,
  KillExpParams, LaunchTensorboardParams, LogsParams, PatchExperimentParams, TrialDetailsParams,
  TrialLogsParams,
} from 'services/types';
import {
  Command, CommandType, Credentials, DeterminedInfo, Experiment, ExperimentDetails, Log,
  TBSourceType, TrialDetails, User,
} from 'types';

/* Helpers */

const saltAndHashPassword = (password?: string): string => {
  if (!password) return '';
  const passwordSalt = 'GubPEmmotfiK9TMD6Zdw';
  return sha512(passwordSalt + password);
};

const commandToEndpoint: Record<CommandType, string> = {
  [CommandType.Command]: '/commands',
  [CommandType.Notebook]: '/notebooks',
  [CommandType.Tensorboard]: '/tensorboard',
  [CommandType.Shell]: '/shells',
};

/* Authentication */

export const getCurrentUser: Api<{}, User> = {
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

/* Info */

export const getInfo: Api<{}, DeterminedInfo> = {
  httpOptions: () => ({ url: '/info' }),
  name: 'getInfo',
  postProcess: (response) => jsonToDeterminedInfo(response.data),
};

/* Commands */

export const killCommand: Api<KillCommandParams, void> = {
  httpOptions: (params) => {
    return {
      method: 'DELETE',
      url: `${commandToEndpoint[params.commandType]}/${params.commandId}`,
    };
  },
  name: 'killCommand',
};

export const launchTensorboard: Api<LaunchTensorboardParams, Command> = {
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
  name: 'launchTensorboard',
  postProcess: (response) => jsonToTensorboard(response.data),
};

/* Experiment */

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
    url: `/trials/${params.id}`,
  }),
  name: 'getTrialDetails',
  postProcess: response => jsonToTrialDetails(response.data),
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

export const getTrialLogs: Api<TrialLogsParams, Log[]> = {
  httpOptions: (params: TrialLogsParams) => ({
    url: [ `/trials/${params.trialId}/logs`, buildQuery(params) ].join('?'),
  }),
  name: 'getTrialLogs',
  postProcess: response => jsonToTrialLogs(response.data),
};

export const getCommandLogs: Api<CommandLogsParams, Log[]> = {
  httpOptions: (params: CommandLogsParams) => ({
    url: [
      `/${params.commandType}/${params.commandId}/events`,
      buildQuery(params),
    ].join('?'),
  }),
  name: 'getCommandLogs',
  postProcess: response => jsonToCommandLogs(response.data),
};
