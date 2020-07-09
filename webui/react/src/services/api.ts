/* eslint-disable @typescript-eslint/camelcase */
import { CancelToken } from 'axios';

import * as DetSwagger from 'services/api-ts-sdk';
import { generateApi, processApiError } from 'services/apiBuilder';
import * as Config from 'services/apiConfig';
import { CommandLogsParams, ExperimentDetailsParams, ExperimentsParams, KillCommandParams,
  KillExpParams, LaunchTensorboardParams, LogsParams, PatchExperimentParams,
  PatchExperimentState,
  TrialLogsParams } from 'services/types';
import {
  AnyTask, CommandType, Credentials, DeterminedInfo, Experiment, ExperimentDetails, Log, User,
} from 'types';
import { serverAddress } from 'utils/routes';
import { isExperimentTask } from 'utils/task';

export const detAuthApi = new DetSwagger.AuthenticationApi(undefined, serverAddress());

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

export const getCurrentUser = generateApi<{}, User>(Config.getCurrentUser);

export const getInfo = generateApi<{}, DeterminedInfo>(Config.getInfo);

export const getExperimentSummaries =
  generateApi<ExperimentsParams, Experiment[]>(Config.getExperimentSummaries);

export const getExperimentDetails =
  generateApi<ExperimentDetailsParams, ExperimentDetails>(Config.getExperimentDetails);

export const killExperiment = generateApi<KillExpParams, void>(Config.killExperiment);

export const killCommand = generateApi<KillCommandParams, void>(Config.killCommand);

export const patchExperiment = generateApi<PatchExperimentParams, void>(Config.patchExperiment);

export const launchTensorboard =
  generateApi<LaunchTensorboardParams, void>(Config.launchTensorboard);

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

export const archiveExperiment = async (
  experimentId: number,
  isArchived: boolean,
  cancelToken?: CancelToken,
): Promise<void> => {
  return await patchExperiment({ body: { archived: isArchived }, cancelToken, experimentId });
};

export const login = generateApi<Credentials, void>(Config.login);

// TODO set up a generic error handler for swagger sdk
// It would be nice to have the input and output types be set automatically
// One this can be achieved is by directly exposing sApi and expecting the user to
// use processApiError.
export function logout(): DetSwagger.V1LogoutResponse {
  const apiName = arguments.callee.name;
  return detAuthApi.determinedLogout().catch(e => processApiError(apiName, e));
}

export const setExperimentState = async (
  { state, ...rest }: PatchExperimentState,
): Promise<void> => {
  return await patchExperiment({
    body: { state },
    ...rest,
  });
};

export const getMasterLogs = generateApi<LogsParams, Log[]>(Config.getMasterLogs);

export const getTrialLogs = generateApi<TrialLogsParams, Log[]>(Config.getTrialLogs);

export const getCommandLogs = generateApi<CommandLogsParams, Log[]>(Config.getCommandLogs);
