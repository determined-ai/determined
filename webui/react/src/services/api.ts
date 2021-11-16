import * as Api from 'services/api-ts-sdk';
import * as Config from 'services/apiConfig';
import * as Service from 'services/types';
import { generateApi, generateDetApi } from 'services/utils';
import * as Type from 'types';
import { terminalCommandStates, tsbMatchesSource } from 'utils/types';

export { isAuthFailure, isLoginFailure, isNotFound } from './utils';

/* Authentication */

export const login = generateDetApi<
  Api.V1LoginRequest, Api.V1LoginResponse, Service.LoginResponse
>(Config.login);

/*
 * Login is an exception where the caller will perform the error handling,
 * so it is one of the few API calls that will not have a try/catch block.
 */
// Temporarily disabling this until we figure out how we want to secure new login endpoint.
// export const login = async (credentials: Credentials): Promise<Api.V1LoginResponse> => {
//   const response = await detApi.Auth.login({
//     password: Config.saltAndHashPassword(credentials.password),
//     username: credentials.username,
//   } as Api.V1LoginRequest);
//   return response;
// };

export const logout = generateDetApi<
  Service.EmptyParams, Api.V1LogoutResponse, void
>(Config.logout);

export const getCurrentUser = generateDetApi<
  Service.EmptyParams, Api.V1CurrentUserResponse, Type.DetailedUser
>(Config.getCurrentUser);

export const getUsers = generateDetApi<
  Service.EmptyParams, Api.V1GetUsersResponse, Type.DetailedUser[]
>(Config.getUsers);

/* Info */

export const getInfo = generateDetApi<
  Service.EmptyParams, Api.V1GetMasterResponse, Type.DeterminedInfo
>(Config.getInfo);

export const getTelemetry = generateDetApi<
  Service.EmptyParams, Api.V1GetTelemetryResponse, Type.Telemetry
>(Config.getTelemetry);

/* Cluster */

export const getAgents = generateDetApi<
  Service.EmptyParams, Api.V1GetAgentsResponse, Type.Agent[]
>(Config.getAgents);

export const getResourcePools = generateDetApi<
  Service.EmptyParams, Api.V1GetResourcePoolsResponse, Type.ResourcePool[]
>(Config.getResourcePools);

export const getResourceAllocationAggregated = generateDetApi<
  Service.GetResourceAllocationAggregatedParams,
  Api.V1ResourceAllocationAggregatedResponse,
  Api.V1ResourceAllocationAggregatedResponse
>(Config.getResourceAllocationAggregated);

/* Experiments */

export const getExperiments = generateDetApi<
  Service.GetExperimentsParams, Api.V1GetExperimentsResponse, Type.ExperimentPagination
>(Config.getExperiments);

export const getExperiment = generateDetApi<
  Service.GetExperimentParams, Api.V1GetExperimentResponse, Type.ExperimentItem
>(Config.getExperiment);

export const getExperimentDetails = generateDetApi<
  Service.ExperimentDetailsParams, Api.V1GetExperimentResponse, Type.ExperimentBase
>(Config.getExperimentDetails);

export const getExpTrials = generateDetApi<
  Service.GetTrialsParams, Api.V1GetExperimentTrialsResponse, Type.TrialPagination
>(Config.getExpTrials);

export const getExpValidationHistory = generateDetApi<
  Service.SingleEntityParams,
  Api.V1GetExperimentValidationHistoryResponse,
  Type.ValidationHistory[]
>(Config.getExpValidationHistory);

export const getTrialDetails = generateDetApi<
  Service.TrialDetailsParams, Api.V1GetTrialResponse, Type.TrialDetails
>(Config.getTrialDetails);

export const createExperiment = generateDetApi<
  Service.CreateExperimentParams, Api.V1CreateExperimentResponse, Type.ExperimentBase
>(Config.createExperiment);

export const archiveExperiment = generateDetApi<
  Service.ExperimentIdParams, Api.V1ArchiveExperimentResponse, void
>(Config.archiveExperiment);

export const unarchiveExperiment = generateDetApi<
  Service.ExperimentIdParams, Api.V1UnarchiveExperimentResponse, void
>(Config.unarchiveExperiment);

export const deleteExperiment = generateDetApi<
  Service.ExperimentIdParams, Api.V1DeleteExperimentResponse, void
>(Config.deleteExperiment);

export const activateExperiment = generateDetApi<
  Service.ExperimentIdParams, Api.V1ActivateExperimentResponse, void
>(Config.activateExperiment);

export const pauseExperiment = generateDetApi<
  Service.ExperimentIdParams, Api.V1PauseExperimentResponse, void
>(Config.pauseExperiment);

export const cancelExperiment = generateDetApi<
  Service.ExperimentIdParams, Api.V1CancelExperimentResponse, void
>(Config.cancelExperiment);

export const killExperiment = generateDetApi<
  Service.ExperimentIdParams, Api.V1KillExperimentResponse, void
>(Config.killExperiment);

export const patchExperiment = generateDetApi<
  Service.PatchExperimentParams, Api.V1KillExperimentResponse, void
>(Config.patchExperiment);

export const getExperimentLabels = generateDetApi<
  Service.EmptyParams, Api.V1GetExperimentLabelsResponse, string[]
>(Config.getExperimentLabels);

/* Tasks */

export const getCommands = generateDetApi<
  Service.GetCommandsParams, Api.V1GetCommandsResponse, Type.CommandTask[]
>(Config.getCommands);

export const getJupyterLabs = generateDetApi<
  Service.GetJupyterLabsParams, Api.V1GetNotebooksResponse, Type.CommandTask[]
>(Config.getJupyterLabs);

export const getShells = generateDetApi<
  Service.GetShellsParams, Api.V1GetShellsResponse, Type.CommandTask[]
>(Config.getShells);

export const getTensorBoards = generateDetApi<
  Service.GetTensorBoardsParams, Api.V1GetTensorboardsResponse, Type.CommandTask[]
>(Config.getTensorBoards);

export const killCommand = generateDetApi<
  Service.CommandIdParams, Api.V1KillCommandResponse, void
>(Config.killCommand);

export const killJupyterLab = generateDetApi<
  Service.CommandIdParams, Api.V1KillNotebookResponse, void
>(Config.killJupyterLab);

export const killShell = generateDetApi<
  Service.CommandIdParams, Api.V1KillShellResponse, void
>(Config.killShell);

export const killTensorBoard = generateDetApi<
  Service.CommandIdParams, Api.V1KillTensorboardResponse, void
>(Config.killTensorBoard);

export const getTaskTemplates = generateDetApi<
  Service.GetTemplatesParams, Api.V1GetTemplatesResponse, Type.Template[]
>(Config.getTemplates);

export const launchJupyterLab = generateDetApi<
  Service.LaunchJupyterLabParams, Api.V1LaunchNotebookResponse, Type.CommandTask
>(Config.launchJupyterLab);

export const previewJupyterLab = generateDetApi<
  Service.LaunchJupyterLabParams, Api.V1LaunchNotebookResponse, Type.RawJson
>(Config.previewJupyterLab);

export const launchTensorBoard = generateDetApi<
  Service.LaunchTensorBoardParams, Api.V1LaunchTensorboardResponse, Type.CommandTask
>(Config.launchTensorBoard);

export const openOrCreateTensorBoard = async (
  params: Service.LaunchTensorBoardParams,
): Promise<Type.CommandTask> => {
  const tensorboards = await getTensorBoards({});
  const match = tensorboards.find(tensorboard =>
    !terminalCommandStates.has(tensorboard.state)
    && tsbMatchesSource(tensorboard, params));
  if (match) return match;
  return launchTensorBoard(params);
};

export const killTask = async (task: Type.CommandTask): Promise<void> => {
  switch (task.type) {
    case Type.CommandType.Command:
      return await killCommand({ commandId: task.id });
    case Type.CommandType.JupyterLab:
      return await killJupyterLab({ commandId: task.id });
    case Type.CommandType.Shell:
      return await killShell({ commandId: task.id });
    case Type.CommandType.TensorBoard:
      return await killTensorBoard({ commandId: task.id });
  }
};

export const getTaskLogs = generateApi<Service.TaskLogsParams, Type.Log[]>(Config.getTaskLogs);
