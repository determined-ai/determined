import { terminalCommandStates } from 'constants/states';
import * as Api from 'services/api-ts-sdk';
import { V1LaunchWarning } from 'services/api-ts-sdk';
import * as Config from 'services/apiConfig';
import * as Service from 'services/types';
import { DeterminedInfo, Telemetry } from 'stores/determinedInfo';
import { EmptyParams, RawJson, SingleEntityParams } from 'types';
import * as Type from 'types';
import { generateDetApi } from 'utils/service';
import { tensorBoardMatchesSource } from 'utils/task';

/* Authentication */

export const login = generateDetApi<Api.V1LoginRequest, Api.V1LoginResponse, Service.LoginResponse>(
  Config.login,
);

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

export const logout = generateDetApi<EmptyParams, Api.V1LogoutResponse, void>(Config.logout);

export const getCurrentUser = generateDetApi<
  EmptyParams,
  Api.V1CurrentUserResponse,
  Type.DetailedUser
>(Config.getCurrentUser);

export const getUser = generateDetApi<
  Service.GetUserParams,
  Api.V1GetUserResponse,
  Type.DetailedUser
>(Config.getUser);

export const postUser = generateDetApi<
  Api.V1PostUserRequest,
  Api.V1PostUserResponse,
  Api.V1PostUserResponse
>(Config.postUser);

export const postUserActivity = generateDetApi<
  Api.V1PostUserActivityRequest,
  Api.V1PostUserActivityResponse,
  Api.V1PostUserActivityResponse
>(Config.postUserActivity);

export const getUsers = generateDetApi<
  Service.GetUsersParams,
  Api.V1GetUsersResponse,
  Type.DetailedUserList
>(Config.getUsers);

export const setUserPassword = generateDetApi<
  Service.SetUserPasswordParams,
  Api.V1SetUserPasswordResponse,
  Api.V1SetUserPasswordResponse
>(Config.setUserPassword);

export const patchUser = generateDetApi<
  Service.PatchUserParams,
  Api.V1PatchUserResponse,
  Type.DetailedUser
>(Config.patchUser);

export const patchUsers = generateDetApi<
  Service.PatchUsersParams,
  Api.V1PatchUsersResponse,
  Api.V1PatchUsersResponse
>(Config.patchUsers);

export const assignMultipleGroups = generateDetApi<
  Api.V1AssignMultipleGroupsRequest,
  Api.V1AssignMultipleGroupsResponse,
  Api.V1AssignMultipleGroupsResponse
>(Config.assignMultipleGroups);

export const getUserSetting = generateDetApi<
  EmptyParams,
  Api.V1GetUserSettingResponse,
  Api.V1GetUserSettingResponse
>(Config.getUserSetting);

export const updateUserSetting = generateDetApi<
  Service.UpdateUserSettingParams,
  Api.V1PostUserSettingResponse,
  void
>(Config.updateUserSetting);

export const resetUserSetting = generateDetApi<EmptyParams, Api.V1ResetUserSettingResponse, void>(
  Config.resetUserSetting,
);

export const getUserPermissions = generateDetApi<
  Service.GetUserParams,
  Api.V1GetPermissionsSummaryResponse,
  Type.PermissionsSummary
>(Config.getUserPermissions);

export const getPermissionsSummary = generateDetApi<
  EmptyParams,
  Api.V1GetPermissionsSummaryResponse,
  Type.PermissionsSummary
>(Config.getPermissionsSummary);

/* Groups */

export const createGroup = generateDetApi<
  Service.CreateGroupsParams,
  Api.V1CreateGroupResponse,
  Api.V1CreateGroupResponse
>(Config.createGroup);

export const getGroup = generateDetApi<
  Service.GetGroupParams,
  Api.V1GetGroupResponse,
  Api.V1GetGroupResponse
>(Config.getGroup);

export const getGroups = generateDetApi<
  Service.GetGroupsParams,
  Api.V1GetGroupsResponse,
  Api.V1GetGroupsResponse
>(Config.getGroups);

export const updateGroup = generateDetApi<
  Service.UpdateGroupParams,
  Api.V1UpdateGroupResponse,
  Api.V1UpdateGroupResponse
>(Config.updateGroup);

export const deleteGroup = generateDetApi<
  Service.DeleteGroupParams,
  Api.V1DeleteGroupResponse,
  Api.V1DeleteGroupResponse
>(Config.deleteGroup);

/* Roles */

export const getGroupRoles = generateDetApi<
  Service.GetGroupParams,
  Api.V1GetRolesAssignedToGroupResponse,
  Type.UserRole[]
>(Config.getGroupRoles);

export const getUserRoles = generateDetApi<
  Service.GetUserParams,
  Api.V1GetRolesAssignedToUserResponse,
  Type.UserRole[]
>(Config.getUserRoles);

export const listRoles = generateDetApi<
  Service.ListRolesParams,
  Api.V1ListRolesResponse,
  Type.UserRole[]
>(Config.listRoles);

export const assignRolesToGroup = generateDetApi<
  Service.AssignRolesToGroupParams[],
  Api.V1AssignRolesResponse,
  Api.V1AssignRolesResponse
>(Config.assignRolesToGroup);

export const removeRolesFromGroup = generateDetApi<
  Service.RemoveRolesFromGroupParams,
  Api.V1RemoveAssignmentsResponse,
  Api.V1RemoveAssignmentsResponse
>(Config.removeRolesFromGroup);

export const assignRolesToUser = generateDetApi<
  Service.AssignRolesToUserParams[],
  Api.V1AssignRolesResponse,
  Api.V1AssignRolesResponse
>(Config.assignRolesToUser);

export const removeRolesFromUser = generateDetApi<
  Service.RemoveRolesFromUserParams,
  Api.V1RemoveAssignmentsResponse,
  Api.V1RemoveAssignmentsResponse
>(Config.removeRolesFromUser);

export const searchRolesAssignableToScope = generateDetApi<
  Service.SearchRolesAssignableToScopeParams,
  Api.V1SearchRolesAssignableToScopeResponse,
  Api.V1SearchRolesAssignableToScopeResponse
>(Config.searchRolesAssignableToScope);
/* Info */

export const getInfo = generateDetApi<EmptyParams, Api.V1GetMasterResponse, DeterminedInfo>(
  Config.getInfo,
);

export const getTelemetry = generateDetApi<EmptyParams, Api.V1GetTelemetryResponse, Telemetry>(
  Config.getTelemetry,
);

/* Cluster */

export const getAgents = generateDetApi<
  Service.GetAgentsParams,
  Api.V1GetAgentsResponse,
  Type.Agent[]
>(Config.getAgents);

export const enableAgent = generateDetApi<string, Api.V1EnableAgentResponse, Type.Agent | null>(
  Config.enableAgent,
);

export const disableAgent = generateDetApi<string, Api.V1DisableAgentResponse, Type.Agent | null>(
  Config.disableAgent,
);

export const getKubernetesResourceManagers = generateDetApi<
  void,
  Api.V1GetKubernetesResourceManagersResponse,
  Type.KubernetesResourceManagers
>(Config.getKubernetesResourceManagers);

export const getResourcePools = generateDetApi<
  Service.GetResourcePoolsParams,
  Api.V1GetResourcePoolsResponse,
  Type.ResourcePool[]
>(Config.getResourcePools);

export const getResourceAllocationAggregated = generateDetApi<
  Service.GetResourceAllocationAggregatedParams,
  Api.V1ResourceAllocationAggregatedResponse,
  Api.V1ResourceAllocationAggregatedResponse
>(Config.getResourceAllocationAggregated);

export const getResourcePoolBindings = generateDetApi<
  Service.GetResourcePoolBindingsParams,
  Api.V1ListWorkspacesBoundToRPResponse,
  Api.V1ListWorkspacesBoundToRPResponse
>(Config.getResourcePoolBindings);

export const deleteResourcePoolBindings = generateDetApi<
  Service.ModifyResourcePoolBindingsParams,
  Api.V1UnbindRPFromWorkspaceResponse,
  void
>(Config.deleteResourcePoolBindings);

export const addResourcePoolBindings = generateDetApi<
  Service.ModifyResourcePoolBindingsParams,
  Api.V1BindRPToWorkspaceResponse,
  void
>(Config.addResourcePoolBindings);

export const overwriteResourcePoolBindings = generateDetApi<
  Service.ModifyResourcePoolBindingsParams,
  Api.V1OverwriteRPWorkspaceBindingsResponse,
  void
>(Config.overwriteResourcePoolBindings);

/* Jobs */

export const getJobQ = generateDetApi<
  Service.GetJobQParams,
  Api.V1GetJobsV2Response,
  Service.GetJobsResponse
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
  Service.GetExperimentsParams,
  Api.V1GetExperimentsResponse,
  Type.ExperimentPagination
>(Config.getExperiments);

export const searchExperiments = generateDetApi<
  Service.SearchExperimentsParams,
  Api.V1SearchExperimentsResponse,
  Type.SearchExperimentPagination
>(Config.searchExperiments);

export const getExperiment = generateDetApi<
  Service.GetExperimentParams,
  Api.V1GetExperimentResponse,
  Type.FullExperimentItem
>(Config.getExperiment);

export const getExperimentDetails = generateDetApi<
  Service.ExperimentDetailsParams,
  Api.V1GetExperimentResponse,
  Type.ExperimentBase
>(Config.getExperimentDetails);

export const getExperimentCheckpoints = generateDetApi<
  Service.getExperimentCheckpointsParams,
  Api.V1GetExperimentCheckpointsResponse,
  Type.CheckpointPagination
>(Config.getExperimentCheckpoints);

export const getExpTrials = generateDetApi<
  Service.GetTrialsParams,
  Api.V1GetExperimentTrialsResponse,
  Type.TrialPagination
>(Config.getExpTrials);

export const getExpValidationHistory = generateDetApi<
  SingleEntityParams,
  Api.V1GetExperimentValidationHistoryResponse,
  Type.ValidationHistory[]
>(Config.getExpValidationHistory);

export const getTrialRemainingLogRetentionDays = generateDetApi<
  Service.TrialDetailsParams,
  Api.V1GetTrialRemainingLogRetentionDaysResponse,
  Type.TrialRemainingLogRetentionDays
>(Config.getTrialRemainingLogRetentionDays);

export const getTrialDetails = generateDetApi<
  Service.TrialDetailsParams,
  Api.V1GetTrialResponse,
  Type.TrialDetails
>(Config.getTrialDetails);

export const timeSeries = generateDetApi<
  Service.TimeSeriesParams,
  Api.V1CompareTrialsResponse,
  Type.TrialSummary[]
>(Config.timeSeries);

export const getTrialCheckpoints = generateDetApi<
  Service.getExperimentCheckpointsParams,
  Api.V1GetTrialCheckpointsResponse,
  Type.CheckpointPagination
>(Config.getTrialCheckpoints);

export const getTrialWorkloads = generateDetApi<
  Service.TrialWorkloadsParams,
  Api.V1GetTrialWorkloadsResponse,
  Type.TrialWorkloads
>(Config.getTrialWorkloads);

export const createExperiment = generateDetApi<
  Service.CreateExperimentParams,
  Api.V1CreateExperimentResponse,
  Type.CreateExperimentResponse
>(Config.createExperiment);

export const continueExperiment = generateDetApi<
  Service.ContinueExperimentParams,
  Api.V1ContinueExperimentResponse,
  void
>(Config.continueExperiment);

export const archiveExperiment = generateDetApi<
  Service.ExperimentIdParams,
  Api.V1ArchiveExperimentResponse,
  void
>(Config.archiveExperiment);

export const archiveExperiments = generateDetApi<
  Service.BulkActionParams,
  Api.V1ArchiveExperimentsResponse,
  Type.BulkActionResult
>(Config.archiveExperiments);

export const unarchiveExperiment = generateDetApi<
  Service.ExperimentIdParams,
  Api.V1UnarchiveExperimentResponse,
  void
>(Config.unarchiveExperiment);

export const unarchiveExperiments = generateDetApi<
  Service.BulkActionParams,
  Api.V1UnarchiveExperimentsResponse,
  Type.BulkActionResult
>(Config.unarchiveExperiments);

export const deleteExperiment = generateDetApi<
  Service.ExperimentIdParams,
  Api.V1DeleteExperimentResponse,
  void
>(Config.deleteExperiment);

export const deleteExperiments = generateDetApi<
  Service.BulkActionParams,
  Api.V1DeleteExperimentsResponse,
  Type.BulkActionResult
>(Config.deleteExperiments);

export const activateExperiment = generateDetApi<
  Service.ExperimentIdParams,
  Api.V1ActivateExperimentResponse,
  void
>(Config.activateExperiment);

export const activateExperiments = generateDetApi<
  Service.BulkActionParams,
  Api.V1ActivateExperimentsResponse,
  Type.BulkActionResult
>(Config.activateExperiments);

export const pauseExperiment = generateDetApi<
  Service.ExperimentIdParams,
  Api.V1PauseExperimentResponse,
  void
>(Config.pauseExperiment);

export const pauseExperiments = generateDetApi<
  Service.BulkActionParams,
  Api.V1PauseExperimentsResponse,
  Type.BulkActionResult
>(Config.pauseExperiments);

export const cancelExperiment = generateDetApi<
  Service.ExperimentIdParams,
  Api.V1CancelExperimentResponse,
  void
>(Config.cancelExperiment);

export const cancelExperiments = generateDetApi<
  Service.BulkActionParams,
  Api.V1CancelExperimentsResponse,
  Type.BulkActionResult
>(Config.cancelExperiments);

export const killExperiment = generateDetApi<
  Service.ExperimentIdParams,
  Api.V1KillExperimentResponse,
  void
>(Config.killExperiment);

export const killExperiments = generateDetApi<
  Service.BulkActionParams,
  Api.V1KillExperimentsResponse,
  Type.BulkActionResult
>(Config.killExperiments);

export const patchExperiment = generateDetApi<
  Service.PatchExperimentParams,
  Api.V1KillExperimentResponse,
  void
>(Config.patchExperiment);

export const getExperimentLabels = generateDetApi<
  Service.ExperimentLabelsParams,
  Api.V1GetExperimentLabelsResponse,
  string[]
>(Config.getExperimentLabels);

export const moveExperiment = generateDetApi<
  Api.V1MoveExperimentRequest,
  Api.V1MoveExperimentResponse,
  void
>(Config.moveExperiment);

export const moveExperiments = generateDetApi<
  Api.V1MoveExperimentsRequest,
  Api.V1MoveExperimentsResponse,
  Type.BulkActionResult
>(Config.moveExperiments);

export const getExperimentFileTree = generateDetApi<
  Service.ExperimentIdParams,
  Api.V1GetModelDefTreeResponse,
  Api.V1FileNode[]
>(Config.getExperimentFileTree);

export const getExperimentFileFromTree = generateDetApi<
  Api.V1GetModelDefFileRequest,
  Api.V1GetModelDefFileResponse,
  string
>(Config.getExperimentFileFromTree);

export const changeExperimentLogRetention = generateDetApi<
  Api.V1PutExperimentsRetainLogsRequest,
  Api.V1PutExperimentsRetainLogsResponse,
  Type.BulkActionResult
>(Config.changeExperimentLogRetention);

/* Tasks */

export const getTask = generateDetApi<
  Service.GetTaskParams,
  Api.V1GetTaskResponse,
  Type.TaskItem | undefined
>(Config.getTask);

export const getActiveTasks = generateDetApi<
  Record<string, never>,
  Api.V1GetActiveTasksCountResponse,
  Type.TaskCounts
>(Config.getActiveTasks);

export const getTaskAcceleratorData = generateDetApi<
  Service.GetTaskParams,
  Api.V1GetTaskAcceleratorDataResponse,
  Api.V1AcceleratorData[]
>(Config.getTaskAcceleratorData);

/* Webhooks */

export const createWebhook = generateDetApi<Api.V1Webhook, Api.V1PostWebhookResponse, Type.Webhook>(
  Config.createWebhook,
);

export const deleteWebhook = generateDetApi<
  Service.GetWebhookParams,
  Api.V1DeleteWebhookResponse,
  void
>(Config.deleteWebhook);

export const getWebhooks = generateDetApi<EmptyParams, Api.V1GetWebhooksResponse, Type.Webhook[]>(
  Config.getWebhooks,
);

export const patchWebhook = generateDetApi<
  Service.PatchWebhookParams,
  Api.V1PatchWebhookResponse,
  void
>(Config.patchWebhook);

export const testWebhook = generateDetApi<
  Service.GetWebhookParams,
  Api.V1TestWebhookResponse,
  void
>(Config.testWebhook);

/* Models */

export const getModels = generateDetApi<
  Service.GetModelsParams,
  Api.V1GetModelsResponse,
  Type.ModelPagination
>(Config.getModels);

export const getModel = generateDetApi<
  Service.GetModelParams,
  Api.V1GetModelResponse,
  Type.ModelItem | undefined
>(Config.getModel);

export const patchModel = generateDetApi<
  Service.PatchModelParams,
  Api.V1PatchModelResponse,
  Type.ModelItem | undefined
>(Config.patchModel);

export const getModelDetails = generateDetApi<
  Service.GetModelDetailsParams,
  Api.V1GetModelVersionsResponse,
  Type.ModelWithVersions | undefined
>(Config.getModelDetails);

export const getModelVersion = generateDetApi<
  Service.GetModelVersionParams,
  Api.V1GetModelVersionResponse,
  Type.ModelVersion | undefined
>(Config.getModelVersion);

export const patchModelVersion = generateDetApi<
  Service.PatchModelVersionParams,
  Api.V1PatchModelVersionResponse,
  Type.ModelVersion | undefined
>(Config.patchModelVersion);

export const archiveModel = generateDetApi<
  Service.ArchiveModelParams,
  Api.V1ArchiveModelResponse,
  void
>(Config.archiveModel);

export const unarchiveModel = generateDetApi<
  Service.ArchiveModelParams,
  Api.V1UnarchiveModelResponse,
  void
>(Config.unarchiveModel);

export const moveModel = generateDetApi<Service.MoveModelParams, Api.V1MoveModelResponse, void>(
  Config.moveModel,
);

export const deleteModel = generateDetApi<
  Service.DeleteModelParams,
  Api.V1DeleteModelResponse,
  void
>(Config.deleteModel);

export const deleteModelVersion = generateDetApi<
  Service.DeleteModelVersionParams,
  Api.V1DeleteModelVersionResponse,
  void
>(Config.deleteModelVersion);

export const postModel = generateDetApi<
  Service.PostModelParams,
  Api.V1PostModelResponse,
  Type.ModelItem | undefined
>(Config.postModel);

export const postModelVersion = generateDetApi<
  Service.PostModelVersionParams,
  Api.V1PostModelVersionResponse,
  Type.ModelVersion | undefined
>(Config.postModelVersion);

export const getModelLabels = generateDetApi<
  Service.GetWorkspaceModelsParams,
  Api.V1GetModelLabelsResponse,
  string[]
>(Config.getModelLabels);

/* Workspaces */

export const getWorkspace = generateDetApi<
  Service.ActionWorkspaceParams,
  Api.V1GetWorkspaceResponse,
  Type.Workspace
>(Config.getWorkspace);

export const getWorkspaces = generateDetApi<
  Service.GetWorkspacesParams,
  Api.V1GetWorkspacesResponse,
  Type.WorkspacePagination
>(Config.getWorkspaces);

export const getWorkspaceMembers = generateDetApi<
  Service.GetWorkspaceMembersParams,
  Api.V1GetGroupsAndUsersAssignedToWorkspaceResponse,
  Type.WorkspaceMembersResponse
>(Config.getWorkspaceMembers);

export const getWorkspaceProjects = generateDetApi<
  Service.GetWorkspaceProjectsParams,
  Api.V1GetWorkspaceProjectsResponse,
  Type.ProjectPagination
>(Config.getWorkspaceProjects);

export const createWorkspace = generateDetApi<
  Api.V1PostWorkspaceRequest,
  Api.V1PostWorkspaceResponse,
  Type.Workspace
>(Config.createWorkspace);

export const deleteWorkspace = generateDetApi<
  Service.ActionWorkspaceParams,
  Api.V1DeleteWorkspaceResponse,
  Type.DeletionStatus
>(Config.deleteWorkspace);

export const patchWorkspace = generateDetApi<
  Service.PatchWorkspaceParams,
  Api.V1PatchWorkspaceResponse,
  Type.Workspace
>(Config.patchWorkspace);

export const listWorkspaceNamespaceBindings = generateDetApi<
  Service.ListWorkspaceNamespaceBindingsParams,
  Api.V1ListWorkspaceNamespaceBindingsResponse,
  Type.WorkspaceNamespaceBindings
>(Config.listWorkspaceNamespaceBindings);

export const setResourceQuotas = generateDetApi<
  Api.V1SetResourceQuotasRequest,
  Api.V1SetResourceQuotasResponse,
  void
>(Config.setResourceQuotas);

export const getKubernetesResourceQuotas = generateDetApi<
  Service.GetKubernetesResourceQuotasParamas,
  Api.V1GetKubernetesResourceQuotasResponse,
  Type.WorkspaceResourceQuotas
>(Config.getKubernetesResourceQuotas);

export const archiveWorkspace = generateDetApi<
  Service.ActionWorkspaceParams,
  Api.V1ArchiveWorkspaceResponse,
  void
>(Config.archiveWorkspace);

export const unarchiveWorkspace = generateDetApi<
  Service.ActionWorkspaceParams,
  Api.V1UnarchiveWorkspaceResponse,
  void
>(Config.unarchiveWorkspace);

export const pinWorkspace = generateDetApi<
  Service.ActionWorkspaceParams,
  Api.V1PinWorkspaceResponse,
  void
>(Config.pinWorkspace);

export const unpinWorkspace = generateDetApi<
  Service.ActionWorkspaceParams,
  Api.V1UnpinWorkspaceResponse,
  void
>(Config.unpinWorkspace);

export const getAvailableResourcePools = generateDetApi<
  Service.ActionWorkspaceParams,
  Api.V1ListRPsBoundToWorkspaceResponse,
  string[]
>(Config.getAvailableResourcePools);

/* Projects */

export const getProject = generateDetApi<
  Service.GetProjectParams,
  Api.V1GetProjectResponse,
  Type.Project
>(Config.getProject);

export const addProjectNote = generateDetApi<
  Service.AddProjectNoteParams,
  Api.V1AddProjectNoteResponse,
  Type.Note[]
>(Config.addProjectNote);

export const setProjectNotes = generateDetApi<
  Service.SetProjectNotesParams,
  Api.V1PutProjectNotesResponse,
  Type.Note[]
>(Config.setProjectNotes);

export const createProject = generateDetApi<
  Api.V1PostProjectRequest,
  Api.V1PostProjectResponse,
  Type.Project
>(Config.createProject);

export const deleteProject = generateDetApi<
  Service.DeleteProjectParams,
  Api.V1DeleteProjectResponse,
  Type.DeletionStatus
>(Config.deleteProject);

export const patchProject = generateDetApi<
  Service.PatchProjectParams,
  Api.V1PatchProjectResponse,
  Type.Project
>(Config.patchProject);

export const moveProject = generateDetApi<
  Api.V1MoveProjectRequest,
  Api.V1MoveProjectResponse,
  void
>(Config.moveProject);

export const archiveProject = generateDetApi<
  Service.ArchiveProjectParams,
  Api.V1ArchiveProjectResponse,
  void
>(Config.archiveProject);

export const unarchiveProject = generateDetApi<
  Service.UnarchiveProjectParams,
  Api.V1UnarchiveProjectResponse,
  void
>(Config.unarchiveProject);

export const getProjectsByUserActivity = generateDetApi<
  Service.GetProjectsByUserActivityParams,
  Api.V1GetProjectsByUserActivityResponse,
  Type.Project[]
>(Config.getProjectsByUserActivity);

export const getProjectColumns = generateDetApi<
  Service.GetProjectColumnsParams,
  Api.V1GetProjectColumnsResponse,
  Type.ProjectColumn[]
>(Config.getProjectColumns);

export const getProjectNumericMetricsRange = generateDetApi<
  Service.GetProjectNumericMetricsRangeParams,
  Api.V1GetProjectNumericMetricsRangeResponse,
  Type.ProjectMetricsRange[]
>(Config.getProjectNumericMetricsRange);

export const getMetadataValues = generateDetApi<
  Service.GetMetadataValuesParams,
  Api.V1GetMetadataValuesResponse,
  string[]
>(Config.getMetadataValues);

/* Runs */

export const searchRuns = generateDetApi<
  Service.SearchRunsParams,
  Api.V1SearchRunsResponse,
  Type.SearchFlatRunPagination
>(Config.searchRuns);

export const archiveRuns = generateDetApi<
  Api.V1ArchiveRunsRequest,
  Api.V1ArchiveRunsResponse,
  Type.BulkActionResult
>(Config.archiveRuns);

export const deleteRuns = generateDetApi<
  Api.V1DeleteRunsRequest,
  Api.V1DeleteRunsResponse,
  Type.BulkActionResult
>(Config.deleteRuns);

export const killRuns = generateDetApi<
  Api.V1KillRunsRequest,
  Api.V1KillRunsResponse,
  Type.BulkActionResult
>(Config.killRuns);

export const moveRuns = generateDetApi<
  Api.V1MoveRunsRequest,
  Api.V1MoveRunsResponse,
  Type.BulkActionResult
>(Config.moveRuns);

export const unarchiveRuns = generateDetApi<
  Api.V1UnarchiveRunsRequest,
  Api.V1UnarchiveRunsResponse,
  void
>(Config.unarchiveRuns);

export const pauseRuns = generateDetApi<
  Api.V1ResumeRunsRequest,
  Api.V1ResumeRunsResponse,
  Type.BulkActionResult
>(Config.pauseRuns);

export const resumeRuns = generateDetApi<
  Api.V1ResumeRunsRequest,
  Api.V1ResumeRunsResponse,
  Type.BulkActionResult
>(Config.resumeRuns);

/* Tasks */

export const getCommands = generateDetApi<
  Service.GetCommandsParams,
  Api.V1GetCommandsResponse,
  Type.CommandTask[]
>(Config.getCommands);

export const getJupyterLabs = generateDetApi<
  Service.GetJupyterLabsParams,
  Api.V1GetNotebooksResponse,
  Type.CommandTask[]
>(Config.getJupyterLabs);

export const getShells = generateDetApi<
  Service.GetShellsParams,
  Api.V1GetShellsResponse,
  Type.CommandTask[]
>(Config.getShells);

export const getTensorBoards = generateDetApi<
  Service.GetTensorBoardsParams,
  Api.V1GetTensorboardsResponse,
  Type.CommandTask[]
>(Config.getTensorBoards);

export const killCommand = generateDetApi<Service.CommandIdParams, Api.V1KillCommandResponse, void>(
  Config.killCommand,
);

export const killJupyterLab = generateDetApi<
  Service.CommandIdParams,
  Api.V1KillNotebookResponse,
  void
>(Config.killJupyterLab);

export const killShell = generateDetApi<Service.CommandIdParams, Api.V1KillShellResponse, void>(
  Config.killShell,
);

export const killTensorBoard = generateDetApi<
  Service.CommandIdParams,
  Api.V1KillTensorboardResponse,
  void
>(Config.killTensorBoard);

export const getJupyterLab = generateDetApi<
  Service.CommandIdParams,
  Api.V1GetNotebookResponse,
  Type.CommandTask
>(Config.getJupyterLab);

export const getShell = generateDetApi<
  Service.CommandIdParams,
  Api.V1GetShellResponse,
  Type.CommandTask
>(Config.getShell);

export const getTensorBoard = generateDetApi<
  Service.CommandIdParams,
  Api.V1GetTensorboardResponse,
  Type.CommandTask
>(Config.getTensorBoard);

export const getCommand = generateDetApi<
  Service.CommandIdParams,
  Api.V1GetCommandResponse,
  Type.CommandTask
>(Config.getCommand);

export const getTaskTemplates = generateDetApi<
  Service.GetTemplatesParams,
  Api.V1GetTemplatesResponse,
  Type.Template[]
>(Config.getTemplates);

export const updateTaskTemplate = generateDetApi<Api.V1Template, Api.V1PutTemplateResponse, void>(
  Config.updateTaskTemplate,
);

export const updateTaskTemplateName = generateDetApi<
  Api.V1PatchTemplateNameRequest,
  Api.V1PatchTemplateNameResponse,
  void
>(Config.updateTaskTemplateName);

export const deleteTaskTemplate = generateDetApi<
  Service.DeleteTemplateParams,
  Api.V1DeleteTemplateResponse,
  void
>(Config.deleteTaskTemplate);

export const createTaskTemplate = generateDetApi<
  Api.V1Template,
  Api.V1PostTemplateResponse,
  Type.Template
>(Config.createTaskTemplate);

export const launchJupyterLab = generateDetApi<
  Service.LaunchJupyterLabParams,
  Api.V1LaunchNotebookResponse,
  Type.CommandResponse
>(Config.launchJupyterLab);

export const previewJupyterLab = generateDetApi<
  Service.LaunchJupyterLabParams,
  Api.V1LaunchNotebookResponse,
  RawJson
>(Config.previewJupyterLab);

export const launchTensorBoard = generateDetApi<
  Service.LaunchTensorBoardParams,
  Api.V1LaunchTensorboardResponse,
  Type.CommandResponse
>(Config.launchTensorBoard);

export const deleteCheckpoints = generateDetApi<
  Api.V1DeleteCheckpointsRequest,
  Api.V1DeleteCheckpointsResponse,
  Api.V1DeleteCheckpointsResponse
>(Config.deleteCheckpoints);

export const openOrCreateTensorBoard = async (
  params: Service.LaunchTensorBoardParams,
): Promise<Type.CommandResponse> => {
  const tensorboards = await getTensorBoards({});
  const match = tensorboards.find(
    (tensorboard) =>
      !terminalCommandStates.has(tensorboard.state) &&
      tensorBoardMatchesSource(tensorboard, params),
  );
  if (match) return { command: match, warnings: [V1LaunchWarning.CURRENTSLOTSEXCEEDED] };
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

export const getWorkspaceConfigPolicies = generateDetApi<
  Service.GetWorkspaceConfigPolicies,
  Api.V1GetWorkspaceConfigPoliciesResponse,
  Api.V1GetWorkspaceConfigPoliciesResponse
>(Config.getWorkspaceConfigPolicies);

export const updateWorkspaceConfigPolicies = generateDetApi<
  Service.UpdateWorkspaceConfigPolicies,
  Api.V1PutWorkspaceConfigPoliciesResponse,
  Api.V1PutWorkspaceConfigPoliciesResponse
>(Config.updateWorkspaceConfigPolicies);
