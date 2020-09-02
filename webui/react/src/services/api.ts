import { CancelToken } from 'axios';

import { TablePagination, TableSorter } from 'components/Table';
import * as DetSwagger from 'services/api-ts-sdk';
import { V1GetExperimentsResponse } from 'services/api-ts-sdk';
import { generateApi, processApiError, serverAddress } from 'services/apiBuilder';
import * as Config from 'services/apiConfig';
import { CreateNotebookParams, CreateTensorboardParams, EmptyParams,
  ExperimentDetailsParams, ExperimentsParams, ForkExperimentParams, KillCommandParams,
  KillExpParams, LogsParams, PatchExperimentParams, PatchExperimentState, TaskLogsParams,
  TrialDetailsParams, TrialLogsParams } from 'services/types';
import {
  Agent, ALL_VALUE, AnyTask, Command, CommandType, Credentials, DetailedUser, DeterminedInfo,
  Experiment, ExperimentDetails, ExperimentFilters, Log, RunState, TrialDetails,
} from 'types';
import { isExperimentTask } from 'utils/task';

import { encodeExperimentState } from './decoder';

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

type ExperimentListSortKey =
  | 'SORT_BY_UNSPECIFIED'
  | 'SORT_BY_ID'
  | 'SORT_BY_DESCRIPTION'
  | 'SORT_BY_START_TIME'
  | 'SORT_BY_END_TIME'
  | 'SORT_BY_STATE'
  | 'SORT_BY_NUM_TRIALS'
  | 'SORT_BY_PROGRESS'
  | 'SORT_BY_USER';

const experimentSortKeys = {
  endTime: 'SORT_BY_END_TIME',
  id: 'SORT_BY_ID',
  name: 'SORT_BY_DESCRIPTION',
  numTrials: 'SORT_BY_NUM_TRIALS',
  progress: 'SORT_BY_PROGRESS',
  startTime: 'SORT_BY_START_TIME',
  state: 'SORT_BY_STATE',
  user: 'SORT_BY_USER',
};
type ExperimentSortKey = keyof typeof experimentSortKeys;

type ExperimentListState =
  | 'STATE_UNSPECIFIED'
  | 'STATE_ACTIVE'
  | 'STATE_PAUSED'
  | 'STATE_STOPPING_COMPLETED'
  | 'STATE_STOPPING_CANCELED'
  | 'STATE_STOPPING_ERROR'
  | 'STATE_COMPLETED'
  | 'STATE_CANCELED'
  | 'STATE_ERROR'
  | 'STATE_DELETED';

export const getExperimentList = async (
  sorter: TableSorter,
  pagination: TablePagination,
  filters: ExperimentFilters,
  search?: string,
): Promise<V1GetExperimentsResponse> => {
  try {
    const response = await detExperimentApi.determinedGetExperiments(
      experimentSortKeys[sorter.key as ExperimentSortKey] as ExperimentListSortKey,
      sorter.descend ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
      pagination.offset,
      pagination.limit,
      search,
      filters.showArchived ? undefined : false,
      filters.states.includes(ALL_VALUE) ? undefined : filters.states.map(state => {
        return encodeExperimentState(state as RunState) as unknown as ExperimentListState;
      }),
      filters.username ? [ filters.username ] : undefined,
    );
    return response;
  } catch (e) {
    processApiError('getExperimentList', e);
    throw e;
  }
};

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
