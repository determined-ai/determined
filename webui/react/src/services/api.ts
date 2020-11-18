import * as Api from 'services/api-ts-sdk';
import * as Config from 'services/apiConfig';
import {
  ApiSorter, CommandIdParams, CreateNotebookParams, CreateTensorboardParams, EmptyParams,
  ExperimentDetailsParams, ExperimentIdParams, ExperimentsParams, ForkExperimentParams,
  LoginResponse, LogsParams, PatchExperimentParams, TaskLogsParams, TrialDetailsParams,
  TrialLogsParams,
} from 'services/types';
import { generateApi, generateDetApi, processApiError } from 'services/utils';
import {
  Agent, ALL_VALUE, Command, CommandTask, CommandType, Credentials, DetailedUser, DeterminedInfo,
  ExperimentBase, ExperimentDetails, ExperimentFilters, ExperimentItem, Log, Pagination, RunState,
  TrialDetails,
} from 'types';
import { terminalCommandStates, tsbMatchesSource } from 'utils/types';

import { decodeExperimentList, encodeExperimentState } from './decoder';

export { isAuthFailure, isLoginFailure, isNotFound } from './utils';

/* Authentication */

export const getCurrentUser =
  generateDetApi<EmptyParams, Api.V1CurrentUserResponse, DetailedUser> (
    Config.getCurrentUser,
  );

export const getUsers = generateApi<EmptyParams, DetailedUser[]>(Config.getUsers);

/* Info */

export const getInfo = generateApi<EmptyParams, DeterminedInfo>(Config.getInfo);

/* Agent */

export const getAgents =
  generateDetApi<EmptyParams, Api.V1GetAgentsResponse, Agent[]> (
    Config.getAgents,
  );

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

export const forkExperiment = generateApi<ForkExperimentParams, number>(Config.forkExperiment);

export const archiveExperiment =
  generateDetApi<ExperimentIdParams, Api.V1ArchiveExperimentResponse, void> (
    Config.archiveExperiment,
  );

export const unarchiveExperiment =
  generateDetApi<ExperimentIdParams, Api.V1UnarchiveExperimentResponse, void> (
    Config.unarchiveExperiment,
  );

export const activateExperiment =
  generateDetApi<ExperimentIdParams, Api.V1ActivateExperimentResponse, void> (
    Config.activateExperiment,
  );

export const pauseExperiment =
  generateDetApi<ExperimentIdParams, Api.V1PauseExperimentResponse, void> (
    Config.pauseExperiment,
  );

export const cancelExperiment =
  generateDetApi<ExperimentIdParams, Api.V1CancelExperimentResponse, void> (
    Config.cancelExperiment,
  );

export const killExperiment =
  generateDetApi<ExperimentIdParams, Api.V1KillExperimentResponse, void> (
    Config.killExperiment,
  );

export const patchExperiment =
  generateDetApi<PatchExperimentParams, Api.V1KillExperimentResponse, void> (
    Config.patchExperiment,
  );

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

export const killCommand =
  generateDetApi<CommandIdParams, Api.V1KillCommandResponse, void> (
    Config.killCommand,
  );

export const killNotebook =
  generateDetApi<CommandIdParams, Api.V1KillNotebookResponse, void> (
    Config.killNotebook,
  );

export const killShell =
  generateDetApi<CommandIdParams, Api.V1KillShellResponse, void> (
    Config.killShell,
  );

export const killTensorboard =
  generateDetApi<CommandIdParams, Api.V1KillTensorboardResponse, void> (
    Config.killTensorboard,
  );

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

export const killTask = async (task: CommandTask): Promise<void> => {
  switch (task.type) {
    case CommandType.Command:
      return await killCommand({ commandId: task.id });
    case CommandType.Notebook:
      return await killNotebook({ commandId: task.id });
    case CommandType.Shell:
      return await killShell({ commandId: task.id });
    case CommandType.Tensorboard:
      return await killTensorboard({ commandId: task.id });
  }
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
    return await Config.detApi.Auth.determinedLogout();
  } catch (e) {
    throw processApiError('logout', e);
  }
};

export const getMasterLogs = generateApi<LogsParams, Log[]>(Config.getMasterLogs);

export const getTaskLogs = generateApi<TaskLogsParams, Log[]>(Config.getTaskLogs);

export const getTrialLogs = generateApi<TrialLogsParams, Log[]>(Config.getTrialLogs);
