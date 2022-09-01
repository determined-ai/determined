import { terminalCommandStates } from 'constants/states';
import * as Api from 'services/api-ts-sdk';
import * as Config from 'services/apiConfig';
import * as Service from 'services/types';
import { EmptyParams, RawJson, SingleEntityParams } from 'shared/types';
import { generateDetApi } from 'shared/utils/service';
import * as Type from 'types';
import { tensorBoardMatchesSource } from 'utils/task';

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
  EmptyParams, Api.V1LogoutResponse, void
>(Config.logout);

export const getCurrentUser = generateDetApi<
  EmptyParams, Api.V1CurrentUserResponse, Type.DetailedUser
>(Config.getCurrentUser);

export const getUser = generateDetApi<
  Service.GetUserParams, Api.V1GetUserResponse, Type.DetailedUser
>(Config.getUser);

export const postUser = generateDetApi<
  Service.PostUserParams, Api.V1PostUserResponse, Api.V1PostUserResponse
>(Config.postUser);

export const getUsers = generateDetApi<
  Service.GetUsersParams,
   Api.V1GetUsersResponse, Type.DetailedUserList
>(Config.getUsers);

export const setUserPassword = generateDetApi<
  Service.SetUserPasswordParams, Api.V1SetUserPasswordResponse, Api.V1SetUserPasswordResponse
>(Config.setUserPassword);

export const patchUser = generateDetApi<
  Service.PatchUserParams, Api.V1PatchUserResponse, Type.DetailedUser
>(Config.patchUser);

export const getUserSetting = generateDetApi<
EmptyParams, Api.V1GetUserSettingResponse, Api.V1GetUserSettingResponse
>(Config.getUserSetting);

export const updateUserSetting = generateDetApi<
  Service.UpdateUserSettingParams, Api.V1PostUserSettingResponse, void
>(Config.updateUserSetting);

export const resetUserSetting = generateDetApi<
  EmptyParams, Api.V1ResetUserSettingResponse, void
>(Config.resetUserSetting);

/* Groups */

export const createGroup = generateDetApi<
  Service.CreateGroupsParams,
   Api.V1CreateGroupResponse, Api.V1CreateGroupResponse
>(Config.createGroup);

export const getGroup = generateDetApi<
  Service.GetGroupParams,
   Api.V1GetGroupResponse, Api.V1GetGroupResponse
>(Config.getGroup);

export const getGroups = generateDetApi<
  Service.GetGroupsParams,
   Api.V1GetGroupsResponse, Api.V1GetGroupsResponse
>(Config.getGroups);

export const updateGroup = generateDetApi<
Service.UpdateGroupParams,
 Api.V1UpdateGroupResponse, Api.V1UpdateGroupResponse
>(Config.updateGroup);

export const deleteGroup = generateDetApi<
Service.DeleteGroupParams,
 Api.V1DeleteGroupResponse, Api.V1DeleteGroupResponse
>(Config.deleteGroup);

/* Info */

export const getInfo = generateDetApi<
  EmptyParams, Api.V1GetMasterResponse, Type.DeterminedInfo
>(Config.getInfo);

export const getTelemetry = generateDetApi<
  EmptyParams, Api.V1GetTelemetryResponse, Type.Telemetry
>(Config.getTelemetry);

/* Cluster */

export const getAgents = generateDetApi<
  EmptyParams, Api.V1GetAgentsResponse, Type.Agent[]
>(Config.getAgents);

export const getResourcePools = generateDetApi<
  EmptyParams, Api.V1GetResourcePoolsResponse, Type.ResourcePool[]
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

export const updateJobQueue = generateDetApi<
  Api.V1UpdateJobQueueRequest,
  Api.V1UpdateJobQueueResponse,
  Api.V1UpdateJobQueueResponse
>(Config.updateJobQueue);

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

export const getExperimentCheckpoints = generateDetApi<
  Service.getExperimentCheckpointsParams,
  Api.V1GetExperimentCheckpointsResponse,
  Type.CheckpointPagination
>(Config.getExperimentCheckpoints);

export const getExpTrials = generateDetApi<
  Service.GetTrialsParams, Api.V1GetExperimentTrialsResponse, Type.TrialPagination
>(Config.getExpTrials);

export const getExpValidationHistory = generateDetApi<
  SingleEntityParams,
  Api.V1GetExperimentValidationHistoryResponse,
  Type.ValidationHistory[]
>(Config.getExpValidationHistory);

export const getTrialDetails = generateDetApi<
  Service.TrialDetailsParams, Api.V1GetTrialResponse, Type.TrialDetails
>(Config.getTrialDetails);

export const compareTrials = generateDetApi<
  Service.CompareTrialsParams, Api.V1CompareTrialsResponse, Type.TrialSummary[]
>(Config.compareTrials);

export const getTrialWorkloads = generateDetApi<
  Service.TrialWorkloadsParams, Api.V1GetTrialWorkloadsResponse, Type.TrialWorkloads
>(Config.getTrialWorkloads);

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
  Service.ExperimentLabelsParams, Api.V1GetExperimentLabelsResponse, string[]
>(Config.getExperimentLabels);

export const moveExperiment = generateDetApi<
  Api.V1MoveExperimentRequest, Api.V1MoveExperimentResponse, void
>(Config.moveExperiment);

export const getExperimentFileTree = generateDetApi<
  Service.ExperimentIdParams, Api.V1GetModelDefTreeResponse, Api.V1FileNode[]
>(Config.getExperimentFileTree);

export const getExperimentFileFromTree = generateDetApi<
  Api.V1GetModelDefFileRequest, Api.V1GetModelDefFileResponse, string
>(Config.getExperimentFileFromTree);

/* Tasks */

export const getTask = generateDetApi<
  Service.GetTaskParams, Api.V1GetTaskResponse, Type.TaskItem | undefined
>(Config.getTask);

export const getActiveTasks = generateDetApi<
  Record<string, never>, Api.V1GetActiveTasksCountResponse, Type.TaskCounts
>(Config.getActiveTasks);

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
  EmptyParams, Api.V1GetModelLabelsResponse, string[]
>(Config.getModelLabels);

/* Workspaces */

export const getWorkspace = generateDetApi<
  Service.GetWorkspaceParams, Api.V1GetWorkspaceResponse, Type.Workspace
>(Config.getWorkspace);

export const getWorkspaces = generateDetApi<
  Service.GetWorkspacesParams, Api.V1GetWorkspacesResponse, Type.WorkspacePagination
>(Config.getWorkspaces);

export const getWorkspaceProjects = generateDetApi<
  Service.GetWorkspaceProjectsParams, Api.V1GetWorkspaceProjectsResponse, Type.ProjectPagination
>(Config.getWorkspaceProjects);

export const createWorkspace = generateDetApi<
  Api.V1PostWorkspaceRequest, Api.V1PostWorkspaceResponse, Type.Workspace
>(Config.createWorkspace);

export const deleteWorkspace = generateDetApi<
  Service.DeleteWorkspaceParams, Api.V1DeleteWorkspaceResponse, Type.DeletionStatus
>(Config.deleteWorkspace);

export const patchWorkspace = generateDetApi<
  Service.PatchWorkspaceParams, Api.V1PatchWorkspaceResponse, Type.Workspace
>(Config.patchWorkspace);

export const archiveWorkspace = generateDetApi<
  Service.ArchiveWorkspaceParams, Api.V1ArchiveWorkspaceResponse, void
>(Config.archiveWorkspace);

export const unarchiveWorkspace = generateDetApi<
  Service.UnarchiveWorkspaceParams, Api.V1UnarchiveWorkspaceResponse, void
>(Config.unarchiveWorkspace);

export const pinWorkspace = generateDetApi<
  Service.PinWorkspaceParams, Api.V1PinWorkspaceResponse, void
>(Config.pinWorkspace);

export const unpinWorkspace = generateDetApi<
  Service.UnpinWorkspaceParams, Api.V1UnpinWorkspaceResponse, void
>(Config.unpinWorkspace);

/* Projects */

export const getProject = generateDetApi<
  Service.GetProjectParams, Api.V1GetProjectResponse, Type.Project
>(Config.getProject);

export const addProjectNote = generateDetApi<
  Service.AddProjectNoteParams, Api.V1AddProjectNoteResponse, Type.Note[]
>(Config.addProjectNote);

export const setProjectNotes = generateDetApi<
  Service.SetProjectNotesParams, Api.V1PutProjectNotesResponse, Type.Note[]
>(Config.setProjectNotes);

export const createProject = generateDetApi<
  Api.V1PostProjectRequest, Api.V1PostProjectResponse, Type.Project
>(Config.createProject);

export const deleteProject = generateDetApi<
  Service.DeleteProjectParams, Api.V1DeleteProjectResponse, Type.DeletionStatus
>(Config.deleteProject);

export const patchProject = generateDetApi<
  Service.PatchProjectParams, Api.V1PatchProjectResponse, Type.Project
>(Config.patchProject);

export const moveProject = generateDetApi<
  Api.V1MoveProjectRequest, Api.V1MoveProjectResponse, void
>(Config.moveProject);

export const archiveProject = generateDetApi<
  Service.ArchiveProjectParams, Api.V1ArchiveProjectResponse, void
>(Config.archiveProject);

export const unarchiveProject = generateDetApi<
  Service.UnarchiveProjectParams, Api.V1UnarchiveProjectResponse, void
>(Config.unarchiveProject);

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
  Service.LaunchJupyterLabParams, Api.V1LaunchNotebookResponse, RawJson
>(Config.previewJupyterLab);

export const launchTensorBoard = generateDetApi<
  Service.LaunchTensorBoardParams, Api.V1LaunchTensorboardResponse, Type.CommandTask
>(Config.launchTensorBoard);

export const openOrCreateTensorBoard = async (
  params: Service.LaunchTensorBoardParams,
): Promise<Type.CommandTask> => {
  const tensorboards = await getTensorBoards({});
  const match = tensorboards.find((tensorboard) =>
    !terminalCommandStates.has(tensorboard.state)
    && tensorBoardMatchesSource(tensorboard, params));
  if (match) return match;
  return launchTensorBoard(params);
};

export const killTask = async (task: Pick<Type.CommandTask, 'id' | 'type'>): Promise<void> => {
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
export { isNotFound } from 'shared/utils/service';
export { isAuthFailure } from 'shared/utils/service';
