import { CancelToken } from 'axios';

import * as Api from 'services/api-ts-sdk';
import * as Config from 'services/apiConfig';
import { ApiSorter, CreateNotebookParams, CreateTensorboardParams, EmptyParams,
  ExperimentDetailsParams, ExperimentsParams, ForkExperimentParams, KillCommandParams,
  KillExpParams, LoginResponse, LogsParams, PatchExperimentParams, PatchExperimentState,
  TaskLogsParams, TrialDetailsParams, TrialLogsParams } from 'services/types';
import { generateApi, generateDetApi, processApiError } from 'services/utils';
import {
  Agent, ALL_VALUE, AnyTask, Command, CommandTask, Credentials,
  DetailedUser, DeterminedInfo, ExperimentBase, ExperimentDetails,
  ExperimentFilters, ExperimentItem, Log, Pagination, RunState, TrialDetails,
} from 'types';
import { isExperimentTask } from 'utils/task';
import { terminalCommandStates, tsbMatchesSource } from 'utils/types';

import { decodeExperimentList, encodeExperimentState } from './decoder';

export { isAuthFailure, isLoginFailure, isNotFound } from './utils';

/* Authentication */

export const getCurrentUser = generateDetApi<EmptyParams, Api.V1CurrentUserResponse, DetailedUser>(
  Config.getCurrentUser,
);

export const getUsers = generateApi<EmptyParams, DetailedUser[]>(Config.getUsers);

/* Info */

export const getInfo = generateApi<EmptyParams, DeterminedInfo>(Config.getInfo);

/* Agent */

export const getAgents = generateApi<EmptyParams, Agent[]>(Config.getAgents);

/* Experiments */

export const getExperimentList = async (
  sorter: ApiSorter<Api.V1GetExperimentsRequestSortBy>,
  pagination: Pagination,
  filters: ExperimentFilters,
  search?: string,
): Promise<{ experiments: ExperimentItem[], pagination?: Api.V1Pagination }> => {
  try {
    const sortBy = Object.values(Api.V1GetExperimentsRequestSortBy).includes(sorter.key) ?
      sorter.key : Api.V1GetExperimentsRequestSortBy.UNSPECIFIED;

    const response = await Config.detApi.Experiments.determinedGetExperiments(
      /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
      sortBy as any,
      sorter.descend ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
      pagination.offset,
      pagination.limit,
      search,
      (filters.labels && filters.labels.length === 0) ? undefined : filters.labels,
      filters.showArchived ? undefined : false,
      filters.states.includes(ALL_VALUE) ? undefined : filters.states.map(state => {
        /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
        return encodeExperimentState(state as RunState) as any;
      }),
      filters.username ? [ filters.username ] : undefined,
    );

    const experiments = decodeExperimentList(response.experiments || []);
    return { experiments, pagination: response.pagination };
  } catch (e) {
    processApiError('getExperimentList', e);
    throw e;
  }
};

export const getExperimentSummaries =
  generateApi<ExperimentsParams, ExperimentBase[]>(Config.getExperimentSummaries);

export const getExperimentDetails =
  generateApi<ExperimentDetailsParams, ExperimentDetails>(Config.getExperimentDetails);

export const getTrialDetails =
  generateApi<TrialDetailsParams, TrialDetails>(Config.getTrialDetails);

export const killExperiment = generateDetApi<KillExpParams, Api.V1KillExperimentResponse, void>(
  Config.killExperiment,
);

export const forkExperiment = generateApi<ForkExperimentParams, number>(Config.forkExperiment);

export const patchExperiment = generateApi<PatchExperimentParams, void>(Config.patchExperiment);

export const archiveExperiment = async (id: number, archive = true): Promise<void> => {
  try {
    await archive ?
      Config.detApi.Experiments.determinedArchiveExperiment(id) :
      Config.detApi.Experiments.determinedUnarchiveExperiment(id);
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

export const getAllExperimentLabels = async (): Promise<string[]> => {
  try {
    const data = await Config.detApi.Experiments.determinedGetExperimentLabels();
    return data.labels || [];
  } catch (e) {
    processApiError('getAllExperimentLabels', e);
    throw e;
  }
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

export const openOrCreateTensorboard = async (
  params: CreateTensorboardParams,
): Promise<Command> => {
  const tensorboards = await getTensorboards({});
  const match = tensorboards.find(tensorboard =>
    !terminalCommandStates.has(tensorboard.state)
    && tsbMatchesSource(tensorboard, params));
  if (match) return match;
  return createTensorboard(params);
};

export const killTask = async (task: AnyTask, cancelToken?: CancelToken): Promise<void> => {
  if (isExperimentTask(task)) {
    return await killExperiment({ cancelToken, experimentId: parseInt(task.id) });
  }
  return await killCommand({
    cancelToken,
    commandId: task.id,
    commandType: (task as CommandTask).type,
  });
};

export const login = generateApi<Credentials, LoginResponse>(Config.login);

/*
 * Login is an exception where the caller will perform the error handling,
 * so it is one of the few API calls that will not have a try/catch block.
 */
// Temporarily disabling this until we figure out how we want to secure new login endpoint.
// export const login = async (credentials: Credentials): Promise<Api.V1LoginResponse> => {
//   const response = await detApi.Auth.determinedLogin({
//     password: Config.saltAndHashPassword(credentials.password),
//     username: credentials.username,
//   } as Api.V1LoginRequest);
//   return response;
// };

export const logout = async (): Promise<Api.V1LogoutResponse> => {
  try {
    const response = await Config.detApi.Auth.determinedLogout();
    return response;
  } catch (e) {
    throw processApiError('logout', e);
  }
};

export const getMasterLogs = generateApi<LogsParams, Log[]>(Config.getMasterLogs);

export const getTaskLogs = generateApi<TaskLogsParams, Log[]>(Config.getTaskLogs);

export const getTrialLogs = generateApi<TrialLogsParams, Log[]>(Config.getTrialLogs);
