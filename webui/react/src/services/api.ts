import { CancelToken } from 'axios';

import * as DetSwagger from 'services/api-ts-sdk';
import { generateApi, processApiError } from 'services/apiBuilder';
import * as Config from 'services/apiConfig';
import { CreateNotebookParams, CreateTensorboardParams, EmptyParams,
  ExperimentDetailsParams, ExperimentsParams, ForkExperimentParams, KillCommandParams,
  KillExpParams, LogsParams, PatchExperimentParams, PatchExperimentState, TaskLogsParams,
  TrialDetailsParams, TrialLogsParams } from 'services/types';
import {
  Agent, AnyTask, Command, CommandType, Credentials, DeterminedInfo, Experiment, ExperimentDetails,
  Log, TrialDetails, User,
} from 'types';
import { serverAddress } from 'utils/routes';
import { isExperimentTask } from 'utils/task';

const address = serverAddress();
export const detAuthApi = new DetSwagger.AuthenticationApi(undefined, address);
export const detExperimentApi = new DetSwagger.ExperimentsApi(undefined, address);
export const detExperimentsStreamingApi = DetSwagger.ExperimentsApiFetchParamCreator();

/* Authentication */

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isAuthFailure = (e: any): boolean => {
  return e.response && e.response.status && e.response.status === 401;
};

// is a failure received from a failed login attempt due to bad credentials
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isLoginFailure = (e: any): boolean => {
  return e.response && e.response.status && e.response.status === 403;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isNotFound = (e: any): boolean => {
  return e.response && e.response.status && e.response.status === 404;
};

export const getCurrentUser = generateApi<EmptyParams, User>(Config.getCurrentUser);

export const getUsers = generateApi<EmptyParams, User[]>(Config.getUsers);

/* Info */

export const getInfo = generateApi<EmptyParams, DeterminedInfo>(Config.getInfo);

/* Agent */

export const getAgents = generateApi<EmptyParams, Agent[]>(Config.getAgents);

/* Experiments */

export const getExperimentSummaries =
  generateApi<ExperimentsParams, Experiment[]>(Config.getExperimentSummaries);

export const getExperimentDetails =
  generateApi<ExperimentDetailsParams, ExperimentDetails>(Config.getExperimentDetails);

export const getTrialDetails =
  generateApi<TrialDetailsParams, TrialDetails>(Config.getTrialDetails);

export const killExperiment = generateApi<KillExpParams, void>(Config.killExperiment);

export const forkExperiment = generateApi<ForkExperimentParams, number>(Config.forkExperiment);

export const patchExperiment = generateApi<PatchExperimentParams, void>(Config.patchExperiment);

export const archiveExperiment = async (id: number, archive = true): Promise<void> => {
  try {
    await archive ?
      detExperimentApi.determinedArchiveExperiment(id) :
      detExperimentApi.determinedUnarchiveExperiment(id);
  } catch (e) {
    processApiError('archiveExperiment', e);
    throw e;
  }
};

export const setExperimentState = async (
  { state, ...rest }: PatchExperimentState,
): Promise<void> => {
  return await patchExperiment({
    body: { state },
    ...rest,
  });
};

/* Tasks */

export const getCommands = generateApi<EmptyParams, Command[]>(Config.getCommands);
export const getNotebooks = generateApi<EmptyParams, Command[]>(Config.getNotebooks);
export const getShells = generateApi<EmptyParams, Command[]>(Config.getShells);
export const getTensorboards = generateApi<EmptyParams, Command[]>(Config.getTensorboards);

export const killCommand = generateApi<KillCommandParams, void>(Config.killCommand);

export const createNotebook = generateApi<CreateNotebookParams, Command>(Config.createNotebook);

export const createTensorboard =
  generateApi<CreateTensorboardParams, Command>(Config.createTensorboard);

export const killTask = async (task: AnyTask, cancelToken?: CancelToken): Promise<void> => {
  if (isExperimentTask(task)) {
    return await killExperiment({ cancelToken, experimentId: parseInt(task.id) });
  }
  return await killCommand({
    cancelToken,
    commandId: task.id,
    commandType: task.type as unknown as CommandType,
  });
};

export const login = generateApi<Credentials, void>(Config.login);

// TODO set up a generic error handler for swagger sdk
// It would be nice to have the input and output types be set automatically
// One this can be achieved is by directly exposing sApi and expecting the user to
// use processApiError.
export const logout = async (): Promise<DetSwagger.V1LogoutResponse> => {
  try {
    const response = await detAuthApi.determinedLogout();
    return response;
  } catch (e) {
    throw processApiError('logout', e);
  }
};

export const getMasterLogs = generateApi<LogsParams, Log[]>(Config.getMasterLogs);

export const getTaskLogs = generateApi<TaskLogsParams, Log[]>(Config.getTaskLogs);

export const getTrialLogs = generateApi<TrialLogsParams, Log[]>(Config.getTrialLogs);
