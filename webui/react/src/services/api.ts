import { terminalCommandStates } from 'constants/states';
import * as Api from 'services/api-ts-sdk';
import * as Config from 'services/apiConfig';
import * as Service from 'services/types';
import { generateApi, generateDetApi } from 'services/utils';
import * as Type from 'types';
import { tensorBoardMatchesSource } from 'utils/task';

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

/* Jobs */

export const getJobQ = generateDetApi<
  Service.GetJobQParams, Api.V1GetJobsResponse, Service.GetJobsResponse
>(Config.getJobQueue);

export const getJobQStats = generateDetApi<
  Service.GetJobQStatsParams,
  Api.V1GetJobQueueStatsResponse,
  Api.V1GetJobQueueStatsResponse
>(Config.getJobQueueStats);

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

export const getTask = generateDetApi<
  Service.GetTaskParams, Api.V1GetTaskResponse, Type.TaskItem | undefined
>(Config.getTask);

/* Models */

export const getModels = generateDetApi<
  Service.GetModelsParams, Api.V1GetModelsResponse, Type.ModelPagination
>(Config.getModels);

export const getModel = generateDetApi<
  Service.GetModelParams, Api.V1GetModelResponse, Type.ModelItem | undefined
>(Config.getModel);

export const patchModel = generateDetApi<
  Service.PatchModelParams, Api.V1PatchModelResponse, Type.ModelItem | undefined
>(Config.patchModel);

export const getModelDetails = generateDetApi<
  Service.GetModelDetailsParams, Api.V1GetModelVersionsResponse, Type.ModelVersions | undefined
>(Config.getModelDetails);

export const getModelVersion = generateDetApi<
  Service.GetModelVersionParams, Api.V1GetModelVersionResponse, Type.ModelVersion | undefined
>(Config.getModelVersion);

export const patchModelVersion = generateDetApi<
  Service.PatchModelVersionParams, Api.V1PatchModelVersionResponse, Type.ModelVersion | undefined
>(Config.patchModelVersion);

export const archiveModel = generateDetApi<
  Service.ArchiveModelParams, Api.V1ArchiveModelResponse, void
>(Config.archiveModel);

export const unarchiveModel = generateDetApi<
  Service.ArchiveModelParams, Api.V1UnarchiveModelResponse, void
>(Config.unarchiveModel);

export const deleteModel = generateDetApi<
  Service.DeleteModelParams, Api.V1DeleteModelResponse, void
>(Config.deleteModel);

export const deleteModelVersion = generateDetApi<
  Service.DeleteModelVersionParams, Api.V1DeleteModelVersionResponse, void
>(Config.deleteModelVersion);

export const postModel = generateDetApi<
  Service.PostModelParams, Api.V1PostModelResponse, Type.ModelItem | undefined
>(Config.postModel);

export const postModelVersion = generateDetApi<
  Service.PostModelVersionParams, Api.V1PostModelVersionResponse, Type.ModelVersion | undefined
>(Config.postModelVersion);

export const getModelLabels = generateDetApi<
  Service.EmptyParams, Api.V1GetModelLabelsResponse, string[]
>(Config.getModelLabels);

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
    && tensorBoardMatchesSource(tensorboard, params));
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
