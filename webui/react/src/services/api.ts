import { CancelToken } from 'axios';
import { sha512 }  from 'js-sha512';

import { decode, ioTypeUser, ioUser } from 'ioTypes';
import { Api, generateApi } from 'services/apiBuilder';
import { jsonToExperiments } from 'services/decoder';
import { CommandType, Credentials, Experiment, RecentTask, TaskType, User } from 'types';

const commandToEndpoint: Record<CommandType, string> = {
  [CommandType.Command]: '/commands',
  [CommandType.Notebook]: '/notebooks',
  [CommandType.Tensorboard]: '/tensorboard',
  [CommandType.Shell]: '/shells',
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isAuthFailure = (e: any): boolean => {
  return e.response && e.response.status && e.response.status === 401;
};

// is a failure received from a failed login attempt due to bad credentials
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isLoginFailure = (e: any): boolean => {
  return e.response && e.response.status && e.response.status === 403;
};

const saltAndHashPassword = (password?: string): string => {
  if (!password) return '';
  const passwordSalt = 'GubPEmmotfiK9TMD6Zdw';
  return sha512(passwordSalt + password);
};

const userApi:  Api<{}, User> = {
  httpOptions: () => { return { url: '/users/me' }; },
  name: 'user',
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

export const getCurrentUser = generateApi<{}, User>(userApi);

export interface ExperimentsParams {
  states?: string[];
}

const experimentsApi:  Api<ExperimentsParams, Experiment[]> = {
  httpOptions: (params) => ({
    url: '/experiment-summaries' + (params.states ? '?states='+params.states.join(',') : ''),
  }),
  name: 'getExperiments',
  postProcess: (response) => jsonToExperiments(response.data),
};

export const getExperiments = generateApi<ExperimentsParams, Experiment[]>(experimentsApi);

interface KillExpParams {
  experimentId: number;
}

const killExperimentApi: Api<KillExpParams, void> = {
  httpOptions: (params) => {
    return {
      method: 'POST',
      url: `/experiments/${params.experimentId.toString()}/kill`,
    };
  },
  name: 'killExperiment',
};

export const killExperiment = generateApi<KillExpParams, void>(killExperimentApi);

interface KillCommandParams {
  commandId: string;
  commandType: CommandType;
}

const killCommandApi: Api<KillCommandParams, void> = {
  httpOptions: (params) => {
    return {
      method: 'DELETE',
      url: `${commandToEndpoint[params.commandType]}/${params.commandId}`,
    };
  },
  name: 'killCommand',
};

export const killCommand = generateApi<KillCommandParams, void>(killCommandApi);

interface PatchExperimentParams {
  experimentId: number;
  body: Record<keyof unknown, unknown> | string;
}

const patchExperimentApi: Api<PatchExperimentParams, void> = {
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

export const patchExperiment = generateApi<PatchExperimentParams, void>(patchExperimentApi);

export const killTask =
  async (task: RecentTask, cancelToken?: CancelToken): Promise<void> => {
    if (task.type === TaskType.Experiment) {
      return killExperiment({ cancelToken, experimentId: parseInt(task.id) });
    }
    return killCommand({
      cancelToken,
      commandId: task.id,
      commandType: task.type as unknown as CommandType,
    });
  };

export const archiveExperiment =
  async (experimentId: number, isArchived: boolean, cancelToken?: CancelToken): Promise<void> => {
    return patchExperiment({ body: { archived: isArchived }, cancelToken, experimentId });
  };

const loginApi: Api<Credentials, void> = {
  httpOptions: ({ password, username }) => {
    return {
      body: { password: saltAndHashPassword(password), username },
      method: 'POST',
      url: '/login?cookie=true',
    };
  },
  name: 'login',
};

export const login = generateApi<Credentials, void>(loginApi);

const logoutApi: Api<{}, void> = {
  httpOptions: () => {
    return {
      method: 'POST',
      url: '/logout',
    };
  },
  name: 'logout',
};

export const logout = generateApi<{}, void>(logoutApi);
