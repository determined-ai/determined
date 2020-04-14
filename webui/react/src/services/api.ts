import { CancelToken } from 'axios';

import { decode, ioTypeUser, ioUser } from 'ioTypes';
import { Api, generateApi } from 'services/apiBuilder';
import { CommandType, RecentTask, TaskType, User } from 'types';

const commandToEndpoint: Record<CommandType, string> = {
  [CommandType.Command]: '/commands',
  [CommandType.Notebook]: '/notebooks',
  [CommandType.Tensorboard]: '/tensorboard',
  [CommandType.Shell]: '/shells',
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

interface KillExpOpts {
  experimentId: number;
}

const killExperimentApi: Api<KillExpOpts, void> = {
  httpOptions: (opts) => {
    return {
      method: 'POST',
      url: `/experiments/${opts.experimentId.toString()}/kill`,
    };
  },
  name: 'killExperiment',
};

export const killExperiment = generateApi<KillExpOpts, void>(killExperimentApi);

interface KillCommandOpts {
  commandId: string;
  commandType: CommandType;
}

const killCommandApi: Api<KillCommandOpts, void> = {
  httpOptions: (opts) => {
    return {
      method: 'DELETE',
      url: `${commandToEndpoint[opts.commandType]}/${opts.commandId}`,
    };
  },
  name: 'killCommand',
};

export const killCommand = generateApi<KillCommandOpts, void>(killCommandApi);

interface PatchExperimentOpts {
  experimentId: number;
  body: Record<keyof unknown, unknown> | string;
}

const patchExperimentApi: Api<PatchExperimentOpts, void> = {
  httpOptions: (opts) => {
    return {
      body: opts.body,
      headers: { 'content-type': 'application/merge-patch+json', 'withCredentials': true },
      method: 'PATCH',
      url: `/experiments/${opts.experimentId.toString()}`,
    };
  },
  name: 'patchExperiment',
};

export const patchExperiment = generateApi<PatchExperimentOpts, void>(patchExperimentApi);

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
