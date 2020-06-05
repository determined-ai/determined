import { sha512 }  from 'js-sha512';

import { decode, ioTypeUser, ioUser } from 'ioTypes';
import { Api } from 'services/apiBuilder';
import { jsonToDeterminedInfo, jsonToExperiments } from 'services/decoder';
import {
  ExperimentsParams, KillCommandParams, KillExpParams,
  LaunchTensorboardParams, PatchExperimentParams,
} from 'services/types';
import { CommandType, Credentials, DeterminedInfo, Experiment, TBSourceType, User } from 'types';

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

export const getCurrentUser:  Api<{}, User> = {
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

export const logout: Api<{}, void> = {
  httpOptions: () => {
    return {
      method: 'POST',
      url: '/logout',
    };
  },
  name: 'logout',
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

export const launchTensorboard: Api<LaunchTensorboardParams, void> = {
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

export const getExperimentSummaries:  Api<ExperimentsParams, Experiment[]> = {
  httpOptions: (params) => ({
    url: '/experiment-summaries' + (params.states ? '?states='+params.states.join(',') : ''),
  }),
  name: 'getExperimentSummaries',
  postProcess: (response) => jsonToExperiments(response.data),
};
