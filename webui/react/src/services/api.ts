import axios, { CancelToken } from 'axios';

import { decode, ioDeterminedInfo, ioTypeDeterminedInfo, ioTypeUser, ioUser } from 'ioTypes';
import { crossoverRoute } from 'routes';
import { CommandType, RecentTask, TaskType, User } from 'types';
import Logger from 'utils/Logger';

const logger = new Logger('api');

export const http = axios.create({
  responseType: 'json',
  withCredentials: true,
});

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
const hasAuthFailed = (e: any): boolean => {
  return e.response && e.response.status && e.response.status === 401;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
const handleAuthFailure = (e: any): boolean => {
  if (!hasAuthFailed(e)) return false;

  // TODO: Update to internal routing when React takes over login.
  crossoverRoute('/ui/logout');
  return true;
};

const commandToEndpoint: Record<CommandType, string> = {
  [CommandType.Command]: '/commands',
  [CommandType.Notebook]: '/notebooks',
  [CommandType.Tensorboard]: '/tensorboard',
  [CommandType.Shell]: '/shells',
};

export const getDeterminedInfo = async (
  cancelToken?: CancelToken,
): Promise<ioTypeDeterminedInfo> => {
  try {
    const response = await http.get('/info', { cancelToken });
    const result = decode<ioTypeDeterminedInfo>(ioDeterminedInfo, response.data);
    return result;
  } catch (e) {
    throw Error('Unable to get platform info.');
  }
};

export const getCurrentUser = async (cancelToken?: CancelToken): Promise<User> => {
  try {
    const response = await http.get('/users/me', { cancelToken });
    const result = decode<ioTypeUser>(ioUser, response.data);
    return {
      id: result.id,
      isActive: result.active,
      isAdmin: result.admin,
      username: result.username,
    };
  } catch (e) {
    handleAuthFailure(e);
    throw Error('Unable to get current user.');
  }
};

export const killExperiment =
  async (experimentId: number, cancelToken?: CancelToken): Promise<void> => {
    try {
      await http.post(`/experiments/${experimentId.toString()}/kill`,
        null, { cancelToken });
    } catch (e) {
      if (!handleAuthFailure(e)) {
        logger.error(e);
        throw Error(`Error during killing experiment ${experimentId}. ${e}`);
      }
    }
  };

export const killCommand =
  async (commandId: string, commandType: CommandType, cancelToken?: CancelToken): Promise<void> => {
    try {
      await http.delete(`${commandToEndpoint[commandType]}/${commandId}`, { cancelToken });
    } catch (e) {
      if (!handleAuthFailure(e)) {
        logger.error(e);
        throw Error(`Error during killing command ${commandId}. ${e}`);
      }
    }
  };

export const killTask =
  async (task: RecentTask, cancelToken?: CancelToken): Promise<void> => {
    if (task.type === TaskType.Experiment) {
      return killExperiment(parseInt(task.id), cancelToken);
    }
    return killCommand(task.id, task.type as unknown as CommandType, cancelToken);
  };

const patchExperiment =
  async (experimentId: number, body: unknown, cancelToken?: CancelToken): Promise<void> => {
    try {
      await axios.patch(`/experiments/${experimentId.toString()}`,
        body, {
          cancelToken,
          headers: { 'content-type': 'application/merge-patch+json', 'withCredentials': true },
        });
    } catch (e) {
      if (!handleAuthFailure(e)) {
        logger.error(e);
        throw Error(`Error during patching experiment ${experimentId}. ${e}`);
      }
    }
  };

export const archiveExperiment =
  async (experimentId: number, isArchived: boolean, cancelToken?: CancelToken): Promise<void> => {
    return patchExperiment(experimentId, { archived: isArchived }, cancelToken);
  };
