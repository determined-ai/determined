import * as Api from 'services/api-ts-sdk';
import * as Config from 'services/apiConfig';
import {
  CommandIdParams, CreateExperimentParams, EmptyParams, ExperimentDetailsParams, ExperimentIdParams,
  GetCommandsParams, GetExperimentParams, GetExperimentsParams, GetJupyterLabsParams,
  GetResourceAllocationAggregatedParams, GetShellsParams, GetTemplatesParams, GetTensorboardsParams,
  GetTrialsParams, LaunchJupyterLabParams, LaunchTensorboardParams, LoginResponse, LogsParams,
  PatchExperimentParams, SingleEntityParams, TaskLogsParams, TrialDetailsParams,
} from 'services/types';
import { generateApi, generateDetApi } from 'services/utils';
import {
  Agent, CommandTask, CommandType, DetailedUser, DeterminedInfo,
  ExperimentBase, ExperimentItem, ExperimentPagination, Log,
  RawJson, ResourcePool, Telemetry, Template,
  TrialDetails, TrialPagination, ValidationHistory,
} from 'types';
import { terminalCommandStates, tsbMatchesSource } from 'utils/types';

export { isAuthFailure, isLoginFailure, isNotFound } from './utils';

/* Authentication */

export const login = generateDetApi<
Api.V1LoginRequest, Api.V1LoginResponse, LoginResponse
>(Config.login);

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

export const logout = generateDetApi<EmptyParams, Api.V1LogoutResponse, void>(Config.logout);

export const getCurrentUser = generateDetApi<
  EmptyParams, Api.V1CurrentUserResponse, DetailedUser
>(Config.getCurrentUser);

export const getUsers = generateDetApi<
  EmptyParams, Api.V1GetUsersResponse, DetailedUser[]
>(Config.getUsers);

/* Info */

export const getInfo = generateDetApi<
  EmptyParams, Api.V1GetMasterResponse, DeterminedInfo
>(Config.getInfo);

export const getTelemetry = generateDetApi<
  EmptyParams, Api.V1GetTelemetryResponse, Telemetry
>(Config.getTelemetry);

/* Cluster */

export const getAgents = generateDetApi<
  EmptyParams, Api.V1GetAgentsResponse, Agent[]
>(Config.getAgents);

export const getResourcePools = generateDetApi<
  EmptyParams, Api.V1GetResourcePoolsResponse, ResourcePool[]
>(Config.getResourcePools);

export const getResourceAllocationAggregated = generateDetApi<
  GetResourceAllocationAggregatedParams, Api.V1ResourceAllocationAggregatedResponse,
  Api.V1ResourceAllocationAggregatedResponse
>(Config.getResourceAllocationAggregated);

/* Experiments */

export const getExperiments = generateDetApi<
  GetExperimentsParams, Api.V1GetExperimentsResponse, ExperimentPagination
>(Config.getExperiments);

export const getExperiment = generateDetApi<
  GetExperimentParams, Api.V1GetExperimentResponse, ExperimentItem
>(Config.getExperiment);

export const getExperimentDetails = generateDetApi<
  ExperimentDetailsParams, Api.V1GetExperimentResponse, ExperimentBase
>(Config.getExperimentDetails);

export const getExpTrials = generateDetApi<
  GetTrialsParams, Api.V1GetExperimentTrialsResponse, TrialPagination
>(Config.getExpTrials);

export const getExpValidationHistory = generateDetApi<
  SingleEntityParams, Api.V1GetExperimentValidationHistoryResponse, ValidationHistory[]
>(Config.getExpValidationHistory);

export const getTrialDetails = generateDetApi<
  TrialDetailsParams, Api.V1GetTrialResponse, TrialDetails
>(Config.getTrialDetails);

export const createExperiment = generateDetApi<
  CreateExperimentParams, Api.V1CreateExperimentResponse, ExperimentBase
>(Config.createExperiment);

export const archiveExperiment = generateDetApi<
  ExperimentIdParams, Api.V1ArchiveExperimentResponse, void
>(Config.archiveExperiment);

export const unarchiveExperiment = generateDetApi<
  ExperimentIdParams, Api.V1UnarchiveExperimentResponse, void
>(Config.unarchiveExperiment);

export const deleteExperiment = generateDetApi<
  ExperimentIdParams, Api.V1DeleteExperimentResponse, void
>(Config.deleteExperiment);

export const activateExperiment = generateDetApi<
  ExperimentIdParams, Api.V1ActivateExperimentResponse, void
>(Config.activateExperiment);

export const pauseExperiment = generateDetApi<
  ExperimentIdParams, Api.V1PauseExperimentResponse, void
>(Config.pauseExperiment);

export const cancelExperiment = generateDetApi<
  ExperimentIdParams, Api.V1CancelExperimentResponse, void
>(Config.cancelExperiment);

export const killExperiment = generateDetApi<
  ExperimentIdParams, Api.V1KillExperimentResponse, void
>(Config.killExperiment);

export const patchExperiment = generateDetApi<
  PatchExperimentParams, Api.V1KillExperimentResponse, void
>(Config.patchExperiment);

export const getExperimentLabels = generateDetApi<
  EmptyParams, Api.V1GetExperimentLabelsResponse, string[]
>(Config.getExperimentLabels);

/* Tasks */

export const getCommands = generateDetApi<
  GetCommandsParams, Api.V1GetCommandsResponse, CommandTask[]
>(Config.getCommands);

export const getJupyterLabs = generateDetApi<
  GetJupyterLabsParams, Api.V1GetNotebooksResponse, CommandTask[]
>(Config.getJupyterLabs);

export const getShells = generateDetApi<
  GetShellsParams, Api.V1GetShellsResponse, CommandTask[]
>(Config.getShells);

export const getTensorboards = generateDetApi<
  GetTensorboardsParams, Api.V1GetTensorboardsResponse, CommandTask[]
>(Config.getTensorboards);

export const killCommand = generateDetApi<
  CommandIdParams, Api.V1KillCommandResponse, void
>(Config.killCommand);

export const killJupyterLab = generateDetApi<
  CommandIdParams, Api.V1KillNotebookResponse, void
>(Config.killJupyterLab);

export const killShell = generateDetApi<
  CommandIdParams, Api.V1KillShellResponse, void
>(Config.killShell);

export const killTensorboard = generateDetApi<
  CommandIdParams, Api.V1KillTensorboardResponse, void
>(Config.killTensorboard);

export const getTaskTemplates = generateDetApi<
GetTemplatesParams, Api.V1GetTemplatesResponse, Template[]
>(Config.getTemplates);

export const launchJupyterLab = generateDetApi<
  LaunchJupyterLabParams, Api.V1LaunchNotebookResponse, CommandTask
>(Config.launchJupyterLab);

export const previewJupyterLab = generateDetApi<
  LaunchJupyterLabParams, Api.V1LaunchNotebookResponse, RawJson
  >(Config.previewJupyterLab);

export const launchTensorboard = generateDetApi<
  LaunchTensorboardParams, Api.V1LaunchTensorboardResponse, CommandTask
>(Config.launchTensorboard);

export const openOrCreateTensorboard = async (
  params: LaunchTensorboardParams,
): Promise<CommandTask> => {
  const tensorboards = await getTensorboards({});
  const match = tensorboards.find(tensorboard =>
    !terminalCommandStates.has(tensorboard.state)
    && tsbMatchesSource(tensorboard, params));
  if (match) return match;
  return launchTensorboard(params);
};

export const killTask = async (task: CommandTask): Promise<void> => {
  switch (task.type) {
    case CommandType.Command:
      return await killCommand({ commandId: task.id });
    case CommandType.JupyterLab:
      return await killJupyterLab({ commandId: task.id });
    case CommandType.Shell:
      return await killShell({ commandId: task.id });
    case CommandType.Tensorboard:
      return await killTensorboard({ commandId: task.id });
  }
};

export const getMasterLogs = generateApi<LogsParams, Log[]>(Config.getMasterLogs);

export const getTaskLogs = generateApi<TaskLogsParams, Log[]>(Config.getTaskLogs);
