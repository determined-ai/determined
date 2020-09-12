import { CancelToken } from 'axios';

import * as DetSwagger from 'services/api-ts-sdk';
import { V1GetExperimentsRequestSortBy, V1Pagination } from 'services/api-ts-sdk';
import { generateApi, processApiError, serverAddress } from 'services/apiBuilder';
import * as Config from 'services/apiConfig';
import { ApiSorter, CreateNotebookParams, CreateTensorboardParams,
  EmptyParams, ExperimentDetailsParams, ExperimentsParams, ForkExperimentParams,
  KillCommandParams, KillExpParams, LogsParams, PatchExperimentParams, PatchExperimentState,
  TaskLogsParams, TrialDetailsParams, TrialLogsParams } from 'services/types';
import {
  Agent, ALL_VALUE, AnyTask, Command, CommandTask, Credentials,
  DetailedUser, DeterminedInfo, ExperimentBase, ExperimentDetails,
  ExperimentFilters, ExperimentItem, Log, Pagination, RunState, TrialDetails,
} from 'types';
import { isExperimentTask } from 'utils/task';

import { decodeExperimentList, encodeExperimentState } from './decoder';

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

export const getCurrentUser = generateApi<EmptyParams, DetailedUser>(Config.getCurrentUser);

export const getUsers = generateApi<EmptyParams, DetailedUser[]>(Config.getUsers);

/* Info */

export const getInfo = generateApi<EmptyParams, DeterminedInfo>(Config.getInfo);

/* Agent */

export const getAgents = generateApi<EmptyParams, Agent[]>(Config.getAgents);

/* Experiments */

export const getExperimentList = async (
  sorter: ApiSorter<V1GetExperimentsRequestSortBy>,
  pagination: Pagination,
  filters: ExperimentFilters,
  search?: string,
): Promise<{ experiments: ExperimentItem[], pagination?: V1Pagination }> => {
  try {
    const sortBy = Object.values(V1GetExperimentsRequestSortBy).includes(sorter.key) ?
      sorter.key : V1GetExperimentsRequestSortBy.UNSPECIFIED;

    const response = await detExperimentApi.determinedGetExperiments(
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
    commandType: (task as CommandTask).type,
  });
};

/*
 * Login is an exception where the caller will perform the error handling,
 * so it is one of the few API calls that will not have a try/catch block.
 */
export const login = async (credentials: Credentials): Promise<DetSwagger.V1LoginResponse> => {
  const response = await detAuthApi.determinedLogin({
    password: Config.saltAndHashPassword(credentials.password),
    username: credentials.username,
  } as DetSwagger.V1LoginRequest);
  return response;
};

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
