import { sha512 } from 'js-sha512';

import { globalStorage } from 'globalStorage';
import { decodeTrialsCollection } from 'pages/TrialsComparison/api';
import { TrialsCollection } from 'pages/TrialsComparison/Collections/collections';
import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';
import * as decoder from 'services/decoder';
import * as Service from 'services/types';
import { DetApi, EmptyParams, RawJson, SingleEntityParams } from 'shared/types';
import { identity, noOp } from 'shared/utils/service';
import { DeterminedInfo, Telemetry } from 'stores/determinedInfo';
import * as Type from 'types';

const updatedApiConfigParams = (
  apiConfig?: Api.ConfigurationParameters,
): Api.ConfigurationParameters => {
  return {
    apiKey: `Bearer ${globalStorage.authToken}`,
    basePath: serverAddress(),
    ...apiConfig,
  };
};

const generateApiConfig = (apiConfig?: Api.ConfigurationParameters) => {
  const config = updatedApiConfigParams(apiConfig);
  return {
    Auth: new Api.AuthenticationApi(config),
    Checkpoint: Api.CheckpointsApiFetchParamCreator(config),
    Cluster: new Api.ClusterApi(config),
    Commands: new Api.CommandsApi(config),
    Experiments: new Api.ExperimentsApi(config),
    Internal: new Api.InternalApi(config),
    Models: new Api.ModelsApi(config),
    Notebooks: new Api.NotebooksApi(config),
    Projects: new Api.ProjectsApi(config),
    RBAC: new Api.RBACApi(config),
    Shells: new Api.ShellsApi(config),
    StreamingCluster: Api.ClusterApiFetchParamCreator(config),
    StreamingExperiments: Api.ExperimentsApiFetchParamCreator(config),
    StreamingInternal: Api.InternalApiFetchParamCreator(config),
    StreamingJobs: Api.JobsApiFetchParamCreator(config),
    StreamingProfiler: Api.ProfilerApiFetchParamCreator(config),
    Tasks: new Api.TasksApi(config),
    Templates: new Api.TemplatesApi(config),
    TensorBoards: new Api.TensorboardsApi(config),
    TrialsComparison: new Api.TrialComparisonApi(config),
    Users: new Api.UsersApi(config),
    Webhooks: new Api.WebhooksApi(config),
    Workspaces: new Api.WorkspacesApi(config),
  };
};

export let detApi = generateApiConfig();

// Update references to generated API code with new configuration.
export const updateDetApi = (apiConfig: Api.ConfigurationParameters): void => {
  detApi = generateApiConfig(apiConfig);
};

/* Helpers */

export const saltAndHashPassword = (password?: string): string => {
  if (!password) return '';
  const passwordSalt = 'GubPEmmotfiK9TMD6Zdw';
  return sha512(passwordSalt + password);
};

export const commandToEndpoint: Record<Type.CommandType, string> = {
  [Type.CommandType.Command]: '/commands',
  [Type.CommandType.JupyterLab]: '/notebooks',
  [Type.CommandType.TensorBoard]: '/tensorboard',
  [Type.CommandType.Shell]: '/shells',
};

export const getUserIds = (users: string[] = []): number[] | undefined => {
  const userIds = users.map((user) => parseInt(user)).filter((user) => !isNaN(user));
  return userIds.length !== 0 ? userIds : undefined;
};

/* Authentication */

export const login: DetApi<Api.V1LoginRequest, Api.V1LoginResponse, Service.LoginResponse> = {
  name: 'login',
  postProcess: (resp) => ({ token: resp.token, user: decoder.mapV1User(resp.user) }),
  request: (params, options) =>
    detApi.Auth.login(
      { ...params, isHashed: true, password: saltAndHashPassword(params.password) },
      options,
    ),
};

export const logout: DetApi<EmptyParams, Api.V1LogoutResponse, void> = {
  name: 'logout',
  postProcess: noOp,
  request: () => detApi.Auth.logout(),
};

export const getCurrentUser: DetApi<EmptyParams, Api.V1CurrentUserResponse, Type.DetailedUser> = {
  name: 'getCurrentUser',
  postProcess: (response) => decoder.mapV1User(response.user),
  // We make sure to request using the latest API configuraitonp parameters.
  request: (options) => detApi.Auth.currentUser(options),
};

export const getUsers: DetApi<
  Service.GetUsersParams,
  Api.V1GetUsersResponse,
  Type.DetailedUserList
> = {
  name: 'getUsers',
  postProcess: (response) => ({
    pagination: decoder.mapV1Pagination(response.pagination),
    users: decoder.mapV1UserList(response),
  }),
  request: (params) =>
    detApi.Users.getUsers(params.sortBy, params.orderBy, params.offset, params.limit, params.name),
};

export const getUser: DetApi<Service.GetUserParams, Api.V1GetUserResponse, Type.DetailedUser> = {
  name: 'getUser',
  postProcess: (response) => decoder.mapV1User(response.user),
  request: (params) => detApi.Users.getUser(params.userId),
};

export const postUser: DetApi<
  Api.V1PostUserRequest,
  Api.V1PostUserResponse,
  Api.V1PostUserResponse
> = {
  name: 'postUser',
  postProcess: (response) => response,
  request: (params) => detApi.Users.postUser(params),
};

export const postUserActivity: DetApi<
  Api.V1PostUserActivityRequest,
  Api.V1PostUserActivityResponse,
  Api.V1PostUserActivityResponse
> = {
  name: 'postUserActivity',
  postProcess: (response) => response,
  request: (params) => detApi.Users.postUserActivity(params),
};

export const setUserPassword: DetApi<
  Service.SetUserPasswordParams,
  Api.V1SetUserPasswordResponse,
  Api.V1SetUserPasswordResponse
> = {
  name: 'setUserPassword',
  postProcess: (response) => response,
  request: (params) => detApi.Users.setUserPassword(params.userId, params.password),
};

export const patchUser: DetApi<
  Service.PatchUserParams,
  Api.V1PatchUserResponse,
  Type.DetailedUser
> = {
  name: 'patchUser',
  postProcess: (response) => decoder.mapV1User(response.user),
  request: (params) => detApi.Users.patchUser(params.userId, params.userParams),
};

export const getUserSetting: DetApi<
  EmptyParams,
  Api.V1GetUserSettingResponse,
  Api.V1GetUserSettingResponse
> = {
  name: 'getUserSetting',
  postProcess: (response) => response,
  request: () => detApi.Users.getUserSetting(),
};

export const updateUserSetting: DetApi<
  Service.UpdateUserSettingParams,
  Api.V1PostUserSettingResponse,
  void
> = {
  name: 'updateUserSetting',
  postProcess: (response) => response,
  request: (params) =>
    detApi.Users.postUserSetting({
      setting: params.setting,
      storagePath: params.storagePath,
    }),
};

export const resetUserSetting: DetApi<
  EmptyParams,
  Api.V1ResetUserSettingResponse,
  Api.V1ResetUserSettingResponse
> = {
  name: 'resetUserSetting',
  postProcess: (response) => response,
  request: () => detApi.Users.resetUserSetting(),
};

/**
 * Returns roles, and workspace/global assignment of those roles,
 * for a user specified in params.
 *
 * @param {GetUserParams} params - An object containing userId to look up their roles.
 */
export const getUserPermissions: DetApi<
  Service.GetUserParams,
  Api.V1GetPermissionsSummaryResponse,
  Type.PermissionsSummary
> = {
  name: 'getUserPermissions',
  postProcess: (response) => ({
    assignments: response.assignments.map(decoder.mapV1UserAssignment),
    roles: response.roles.map(decoder.mapV1Role),
  }),
  request: (params) => detApi.RBAC.getPermissionsSummary(params.userId),
};

/**
 * Returns roles, and workspace/global assignment of the roles,
 * associated with the active/requesting user.
 */
export const getPermissionsSummary: DetApi<
  EmptyParams,
  Api.V1GetPermissionsSummaryResponse,
  Type.PermissionsSummary
> = {
  name: 'getPermissionsSummary',
  postProcess: (response) => ({
    assignments: response.assignments.map(decoder.mapV1UserAssignment),
    roles: response.roles.map(decoder.mapV1Role),
  }),
  request: () => detApi.RBAC.getPermissionsSummary(),
};

/* Group */

export const createGroup: DetApi<
  Service.CreateGroupsParams,
  Api.V1CreateGroupResponse,
  Api.V1CreateGroupResponse
> = {
  name: 'createGroup',
  postProcess: (response) => response,
  request: (params) =>
    detApi.Internal.createGroup({
      addUsers: params.addUsers,
      name: params.name,
    }),
};

export const getGroup: DetApi<
  Service.GetGroupParams,
  Api.V1GetGroupResponse,
  Api.V1GetGroupResponse
> = {
  name: 'getGroup',
  postProcess: (response) => response,
  request: (params) => detApi.Internal.getGroup(params.groupId),
};

export const getGroups: DetApi<
  Service.GetGroupsParams,
  Api.V1GetGroupsResponse,
  Api.V1GetGroupsResponse
> = {
  name: 'getGroups',
  postProcess: (response) => response,
  request: (params) =>
    detApi.Internal.getGroups({
      limit: params.limit || 10,
      offset: params.offset,
      userId: params.userId,
    }),
};

export const updateGroup: DetApi<
  Service.UpdateGroupParams,
  Api.V1UpdateGroupResponse,
  Api.V1UpdateGroupResponse
> = {
  name: 'updateGroup',
  postProcess: (response) => response,
  request: (params) =>
    detApi.Internal.updateGroup(params.groupId, {
      addUsers: params.addUsers,
      groupId: params.groupId,
      name: params.name,
      removeUsers: params.removeUsers,
    }),
};

export const deleteGroup: DetApi<
  Service.DeleteGroupParams,
  Api.V1DeleteGroupResponse,
  Api.V1DeleteGroupResponse
> = {
  name: 'deleteGroup',
  postProcess: (response) => response,
  request: (params) => detApi.Internal.deleteGroup(params.groupId),
};

/* Roles */

const ROLES_LIMIT = 1000;

export const getGroupRoles: DetApi<
  Service.GetGroupParams,
  Api.V1GetRolesAssignedToGroupResponse,
  Type.UserRole[]
> = {
  name: 'getRolesAssignedToGroup',
  postProcess: (response) => decoder.mapV1GroupRole(response),
  request: (params) => detApi.RBAC.getRolesAssignedToGroup(params.groupId),
};

export const getUserRoles: DetApi<
  Service.GetUserParams,
  Api.V1GetRolesAssignedToUserResponse,
  Type.UserRole[]
> = {
  name: 'getRolesAssignedToUser',
  postProcess: (response) => response.roles.map((rwa) => decoder.mapV1UserRole(rwa)),
  request: (params) => detApi.RBAC.getRolesAssignedToUser(params.userId),
};

export const listRoles: DetApi<Service.ListRolesParams, Api.V1ListRolesResponse, Type.UserRole[]> =
  {
    name: 'listRoles',
    postProcess: (response) => response.roles.map(decoder.mapV1Role),
    request: (params) =>
      detApi.RBAC.listRoles({
        limit: params.limit || 0,
        offset: params.offset || 0,
      }),
  };

export const assignRolesToGroup: DetApi<
  Service.AssignRolesToGroupParams,
  Api.V1AssignRolesResponse,
  Api.V1AssignRolesResponse
> = {
  name: 'assignRolesToGroup',
  postProcess: (response) => response,
  request: (params) =>
    detApi.RBAC.assignRoles({
      groupRoleAssignments: params.roleIds.map((roleId) => ({
        groupId: params.groupId,
        roleAssignment: {
          role: { roleId },
          scopeWorkspaceId: params.scopeWorkspaceId || undefined,
        },
      })),
    }),
};

export const removeRolesFromGroup: DetApi<
  Service.RemoveRolesFromGroupParams,
  Api.V1RemoveAssignmentsResponse,
  Api.V1RemoveAssignmentsResponse
> = {
  name: 'removeRolesFromGroup',
  postProcess: (response) => response,
  request: (params) =>
    detApi.RBAC.removeAssignments({
      groupRoleAssignments: params.roleIds.map((roleId) => ({
        groupId: params.groupId,
        roleAssignment: {
          role: { roleId },
          scopeWorkspaceId: params.scopeWorkspaceId || undefined,
        },
      })),
    }),
};

export const assignRolesToUser: DetApi<
  Service.AssignRolesToUserParams,
  Api.V1AssignRolesResponse,
  Api.V1AssignRolesResponse
> = {
  name: 'assignRolesToUser',
  postProcess: (response) => response,
  request: (params) =>
    detApi.RBAC.assignRoles({
      userRoleAssignments: params.roleIds.map((roleId) => ({
        roleAssignment: {
          role: { roleId },
          scopeWorkspaceId: params.scopeWorkspaceId || undefined,
        },
        userId: params.userId,
      })),
    }),
};

export const removeRolesFromUser: DetApi<
  Service.RemoveRolesFromUserParams,
  Api.V1RemoveAssignmentsResponse,
  Api.V1RemoveAssignmentsResponse
> = {
  name: 'removeRolesFromUser',
  postProcess: (response) => response,
  request: (params) =>
    detApi.RBAC.removeAssignments({
      userRoleAssignments: params.roleIds.map((roleId) => ({
        roleAssignment: {
          role: { roleId },
          scopeWorkspaceId: params.scopeWorkspaceId || undefined,
        },
        userId: params.userId,
      })),
    }),
};

export const searchRolesAssignableToScope: DetApi<
  Service.SearchRolesAssignableToScopeParams,
  Api.V1SearchRolesAssignableToScopeResponse,
  Api.V1SearchRolesAssignableToScopeResponse
> = {
  name: 'searchRolesAssignableToScope',
  postProcess: (response) => response,
  request: (params) =>
    detApi.RBAC.searchRolesAssignableToScope({
      limit: params.limit || ROLES_LIMIT,
      offset: params.offset,
      workspaceId: params.workspaceId,
    }),
};

/* Info */

export const getInfo: DetApi<EmptyParams, Api.V1GetMasterResponse, DeterminedInfo> = {
  name: 'getInfo',
  postProcess: (response) => decoder.mapV1MasterInfo(response),
  request: () => detApi.Cluster.getMaster(),
};

export const getTelemetry: DetApi<EmptyParams, Api.V1GetTelemetryResponse, Telemetry> = {
  name: 'getTelemetry',
  postProcess: (response) => response,
  request: () => detApi.Internal.getTelemetry(),
};

/* Cluster */

export const getAgents: DetApi<EmptyParams, Api.V1GetAgentsResponse, Type.Agent[]> = {
  name: 'getAgents',
  postProcess: (response) => decoder.jsonToAgents(response.agents || []),
  request: () => detApi.Cluster.getAgents(),
};

export const getResourcePools: DetApi<
  EmptyParams,
  Api.V1GetResourcePoolsResponse,
  Type.ResourcePool[]
> = {
  name: 'getResourcePools',
  postProcess: (response) => {
    return response.resourcePools?.map(decoder.mapV1ResourcePool) || [];
  },
  request: () => detApi.Internal.getResourcePools(),
};

export const getResourceAllocationAggregated: DetApi<
  Service.GetResourceAllocationAggregatedParams,
  Api.V1ResourceAllocationAggregatedResponse,
  Api.V1ResourceAllocationAggregatedResponse
> = {
  name: 'getResourceAllocationAggregated',
  postProcess: (response) => response,
  request: (params: Service.GetResourceAllocationAggregatedParams, options) => {
    const dateFormat =
      params.period === 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY' ? 'YYYY-MM' : 'YYYY-MM-DD';
    return detApi.Cluster.resourceAllocationAggregated(
      params.startDate.format(dateFormat),
      params.endDate.format(dateFormat),
      params.period,
      options,
    );
  },
};

/* Trials */
export const queryTrials: DetApi<
  Api.V1QueryTrialsRequest,
  Api.V1QueryTrialsResponse,
  Api.V1QueryTrialsResponse
> = {
  name: 'queryTrials',
  postProcess: (response: Api.V1QueryTrialsResponse): Api.V1QueryTrialsResponse => {
    return response;
  },
  request: (params: Api.V1QueryTrialsRequest) => {
    return detApi.TrialsComparison.queryTrials({
      ...params,
      limit: params?.limit ? 3 * params.limit : 30,
    });
  },
};

export const updateTrialTags: DetApi<
  Api.V1UpdateTrialTagsRequest,
  Api.V1UpdateTrialTagsResponse,
  Api.V1UpdateTrialTagsResponse
> = {
  name: 'updateTrialTags',
  postProcess: (response: Api.V1UpdateTrialTagsResponse) => {
    return { rowsAffected: response.rowsAffected };
  },
  request: (params: Api.V1UpdateTrialTagsRequest) => {
    return detApi.TrialsComparison.updateTrialTags(params);
  },
};

export const createTrialCollection: DetApi<
  Api.V1CreateTrialsCollectionRequest,
  Api.V1CreateTrialsCollectionResponse,
  TrialsCollection | undefined
> = {
  name: 'createTrialsCollection',
  postProcess: (response: Api.V1CreateTrialsCollectionResponse) =>
    response.collection ? decodeTrialsCollection(response.collection) : undefined,
  request: (params: Api.V1CreateTrialsCollectionRequest) => {
    return detApi.TrialsComparison.createTrialsCollection(params);
  },
};

export const getTrialsCollections: DetApi<
  number,
  Api.V1GetTrialsCollectionsResponse,
  Api.V1GetTrialsCollectionsResponse
> = {
  name: 'getTrialsCollection',
  postProcess: (response: Api.V1GetTrialsCollectionsResponse) => {
    return { collections: response.collections };
  },
  request: (projectId: number) => {
    return detApi.TrialsComparison.getTrialsCollections(projectId);
  },
};

export const patchTrialsCollection: DetApi<
  Api.V1PatchTrialsCollectionRequest,
  Api.V1PatchTrialsCollectionResponse,
  TrialsCollection | undefined
> = {
  name: 'patchTrialsCollection',
  postProcess: (response: Api.V1PatchTrialsCollectionResponse) =>
    response.collection ? decodeTrialsCollection(response.collection) : undefined,
  request: (params: Api.V1PatchTrialsCollectionRequest) => {
    return detApi.TrialsComparison.patchTrialsCollection(params);
  },
};

export const deleteTrialsCollection: DetApi<
  number,
  Api.V1DeleteTrialsCollectionResponse,
  Api.V1DeleteTrialsCollectionResponse
> = {
  name: 'deleteTrialsCollection',
  postProcess: (response: Api.V1DeleteTrialsCollectionResponse) => response,
  request: (id: number) => {
    return detApi.TrialsComparison.deleteTrialsCollection(id);
  },
};

/* Experiment */

export const getExperiments: DetApi<
  Service.GetExperimentsParams,
  Api.V1GetExperimentsResponse,
  Type.ExperimentPagination
> = {
  name: 'getExperiments',
  postProcess: (response: Api.V1GetExperimentsResponse) => {
    return {
      experiments: decoder.mapV1ExperimentList(response.experiments),
      pagination: response.pagination,
    };
  },
  request: (params: Service.GetExperimentsParams, options) => {
    return detApi.Experiments.getExperiments(
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      params.description,
      params.name,
      params.labels,
      params.archived,
      params.states,
      undefined,
      getUserIds(params.users),
      params.projectId ?? 0,
      undefined,
      undefined,
      undefined,
      undefined,
      params.experimentIdFilter?.incl,
      params.experimentIdFilter?.notIn,
      true,
      options,
    );
  },
};

export const searchExperiments: DetApi<
  Service.SearchExperimentsParams,
  Api.V1SearchExperimentsResponse,
  Type.SearchExperimentPagination
> = {
  name: 'searchExperiments',
  postProcess: (response) => {
    return {
      experiments: response.experiments.map((e) => decoder.mapSearchExperiment(e)),
      pagination: response.pagination,
    };
  },
  request: (params: Service.SearchExperimentsParams, options) => {
    return detApi.Experiments.searchExperiments(
      params.projectId,
      params.offset,
      params.limit,
      params.sort,
      options,
    );
  },
};

export const getExperiment: DetApi<
  Service.GetExperimentParams,
  Api.V1GetExperimentResponse,
  Type.ExperimentItem
> = {
  name: 'getExperiment',
  postProcess: (response: Api.V1GetExperimentResponse) => {
    const exp = decoder.mapV1Experiment(response.experiment);
    exp.jobSummary = response.jobSummary;
    return exp;
  },
  request: (params: Service.GetExperimentParams) => {
    return detApi.Experiments.getExperiment(params.id);
  },
};

export const createExperiment: DetApi<
  Service.CreateExperimentParams,
  Api.V1CreateExperimentResponse,
  Type.CreateExperimentResponse
> = {
  name: 'createExperiment',
  postProcess: (resp: Api.V1CreateExperimentResponse) => {
    return {
      experiment: decoder.mapV1GetExperimentDetailsResponse(resp),
      warnings: resp.warnings || [],
    };
  },
  request: (params: Service.CreateExperimentParams, options) => {
    return detApi.Internal.createExperiment(
      {
        activate: params.activate,
        config: params.experimentConfig,
        parentId: params.parentId,
        projectId: params.projectId,
      },
      options,
    );
  },
};

export const archiveExperiment: DetApi<
  Service.ExperimentIdParams,
  Api.V1ArchiveExperimentResponse,
  void
> = {
  name: 'archiveExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.archiveExperiment(params.experimentId, options);
  },
};

export const archiveExperiments: DetApi<
  Service.BulkActionParams,
  Api.V1ArchiveExperimentsResponse,
  Type.BulkActionResult
> = {
  name: 'archiveExperiments',
  postProcess: (response) => decoder.mapV1ExperimentActionResults(response.results),
  request: (params: Service.BulkActionParams, options) => {
    return detApi.Experiments.archiveExperiments(params, options);
  },
};

export const deleteExperiment: DetApi<
  Service.ExperimentIdParams,
  Api.V1DeleteExperimentResponse,
  void
> = {
  name: 'deleteExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.deleteExperiment(params.experimentId, options);
  },
};

export const deleteExperiments: DetApi<
  Service.BulkActionParams,
  Api.V1DeleteExperimentsResponse,
  Type.BulkActionResult
> = {
  name: 'deleteExperiments',
  postProcess: (response) => decoder.mapV1ExperimentActionResults(response.results),
  request: (params: Service.BulkActionParams, options) => {
    return detApi.Experiments.deleteExperiments(params, options);
  },
};

export const unarchiveExperiment: DetApi<
  Service.ExperimentIdParams,
  Api.V1UnarchiveExperimentResponse,
  void
> = {
  name: 'unarchiveExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.unarchiveExperiment(params.experimentId, options);
  },
};

export const unarchiveExperiments: DetApi<
  Service.BulkActionParams,
  Api.V1UnarchiveExperimentsResponse,
  Type.BulkActionResult
> = {
  name: 'unarchiveExperiments',
  postProcess: (response) => decoder.mapV1ExperimentActionResults(response.results),
  request: (params: Service.BulkActionParams, options) => {
    return detApi.Experiments.unarchiveExperiments(params, options);
  },
};

export const activateExperiment: DetApi<
  Service.ExperimentIdParams,
  Api.V1ActivateExperimentResponse,
  void
> = {
  name: 'activateExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.activateExperiment(params.experimentId, options);
  },
};

export const activateExperiments: DetApi<
  Service.BulkActionParams,
  Api.V1ActivateExperimentsResponse,
  Type.BulkActionResult
> = {
  name: 'activateExperiments',
  postProcess: (response) => decoder.mapV1ExperimentActionResults(response.results),
  request: (params: Service.BulkActionParams, options) => {
    return detApi.Experiments.activateExperiments(params, options);
  },
};

export const pauseExperiment: DetApi<
  Service.ExperimentIdParams,
  Api.V1PauseExperimentResponse,
  void
> = {
  name: 'pauseExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.pauseExperiment(params.experimentId, options);
  },
};

export const pauseExperiments: DetApi<
  Service.BulkActionParams,
  Api.V1PauseExperimentsResponse,
  Type.BulkActionResult
> = {
  name: 'pauseExperiments',
  postProcess: (response) => decoder.mapV1ExperimentActionResults(response.results),
  request: (params: Service.BulkActionParams, options) => {
    return detApi.Experiments.pauseExperiments(params, options);
  },
};

export const cancelExperiment: DetApi<
  Service.ExperimentIdParams,
  Api.V1CancelExperimentResponse,
  void
> = {
  name: 'cancelExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.cancelExperiment(params.experimentId, options);
  },
};

export const cancelExperiments: DetApi<
  Service.BulkActionParams,
  Api.V1CancelExperimentsResponse,
  Type.BulkActionResult
> = {
  name: 'cancelExperiments',
  postProcess: (response) => decoder.mapV1ExperimentActionResults(response.results),
  request: (params: Service.BulkActionParams, options) => {
    return detApi.Experiments.cancelExperiments(params, options);
  },
};

export const killExperiment: DetApi<
  Service.ExperimentIdParams,
  Api.V1KillExperimentResponse,
  void
> = {
  name: 'killExperiment',
  postProcess: noOp,
  request: (params: Service.ExperimentIdParams, options) => {
    return detApi.Experiments.killExperiment(params.experimentId, options);
  },
};

export const killExperiments: DetApi<
  Service.BulkActionParams,
  Api.V1KillExperimentsResponse,
  Type.BulkActionResult
> = {
  name: 'killExperiments',
  postProcess: (response) => decoder.mapV1ExperimentActionResults(response.results),
  request: (params: Service.BulkActionParams, options) => {
    return detApi.Experiments.killExperiments(params, options);
  },
};

export const patchExperiment: DetApi<
  Service.PatchExperimentParams,
  Api.V1PatchExperimentResponse,
  void
> = {
  name: 'patchExperiment',
  postProcess: noOp,
  request: (params: Service.PatchExperimentParams, options) => {
    return detApi.Experiments.patchExperiment(
      params.experimentId,
      params.body as Api.V1Experiment,
      options,
    );
  },
};

export const getExperimentDetails: DetApi<
  Service.ExperimentDetailsParams,
  Api.V1GetExperimentResponse,
  Type.ExperimentBase
> = {
  name: 'getExperimentDetails',
  postProcess: (response) => decoder.mapV1GetExperimentDetailsResponse(response),
  request: (params, options) => detApi.Experiments.getExperiment(params.id, options),
};

export const getExperimentCheckpoints: DetApi<
  Service.getExperimentCheckpointsParams,
  Api.V1GetExperimentCheckpointsResponse,
  Type.CheckpointPagination
> = {
  name: 'getExperimentCheckpoints',
  postProcess: (response) => decoder.decodeCheckpoints(response),
  request: (params, options) =>
    detApi.Experiments.getExperimentCheckpoints(
      params.id,
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      params.states,
      options,
    ),
};

export const getExpValidationHistory: DetApi<
  SingleEntityParams,
  Api.V1GetExperimentValidationHistoryResponse,
  Type.ValidationHistory[]
> = {
  name: 'getExperimentValidationHistory',
  postProcess: (response) => {
    if (!response.validationHistory) return [];
    return response.validationHistory?.map((vh) => ({
      endTime: vh.endTime as unknown as string,
      trialId: vh.trialId,
      validationError: vh.searcherMetric,
    }));
  },
  request: (params, options) => {
    return detApi.Experiments.getExperimentValidationHistory(params.id, options);
  },
};

export const getExpTrials: DetApi<
  Service.GetTrialsParams,
  Api.V1GetExperimentTrialsResponse,
  Type.TrialPagination
> = {
  name: 'getExperimentTrials',
  postProcess: (response) => {
    return {
      pagination: response.pagination,
      trials: response.trials.map(decoder.decodeV1TrialToTrialItem),
    };
  },
  request: (params, options) => {
    return detApi.Experiments.getExperimentTrials(
      params.id,
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      params.states,
      options,
    );
  },
};

export const getExperimentLabels: DetApi<
  Service.ExperimentLabelsParams,
  Api.V1GetExperimentLabelsResponse,
  string[]
> = {
  name: 'getExperimentLabels',
  postProcess: (response) => response.labels || [],
  request: (params, options) => detApi.Experiments.getExperimentLabels(params.project_id, options),
};

export const getTrialDetails: DetApi<
  Service.TrialDetailsParams,
  Api.V1GetTrialResponse,
  Type.TrialDetails
> = {
  name: 'getTrialDetails',
  postProcess: (response: Api.V1GetTrialResponse) => {
    return decoder.decodeTrialResponseToTrialDetails(response);
  },
  request: (params: Service.TrialDetailsParams) => detApi.Experiments.getTrial(params.id),
};

export const moveExperiment: DetApi<
  Api.V1MoveExperimentRequest,
  Api.V1MoveExperimentResponse,
  void
> = {
  name: 'moveExperiment',
  postProcess: noOp,
  request: (params) =>
    detApi.Experiments.moveExperiment(params.experimentId, {
      destinationProjectId: params.destinationProjectId,
      experimentId: params.experimentId,
    }),
};

export const moveExperiments: DetApi<
  Api.V1MoveExperimentsRequest,
  Api.V1MoveExperimentsResponse,
  Type.BulkActionResult
> = {
  name: 'moveExperiments',
  postProcess: (response) => decoder.mapV1ExperimentActionResults(response.results),
  request: (params, options) => detApi.Experiments.moveExperiments(params, options),
};

export const timeSeries: DetApi<
  Service.TimeSeriesParams,
  Api.V1CompareTrialsResponse,
  Type.TrialSummary[]
> = {
  name: 'timeSeries',
  postProcess: (response: Api.V1CompareTrialsResponse) => {
    return response.trials.map(decoder.decodeTrialSummary);
  },
  request: (params: Service.TimeSeriesParams) =>
    detApi.Experiments.compareTrials(
      params.trialIds,
      params.maxDatapoints,
      params.metricNames.map((m) => m.name),
      params.startBatches,
      params.endBatches,
      params.metricType ? Type.metricTypeParamMap[params.metricType] : 'METRIC_TYPE_UNSPECIFIED',
      params.scale === Type.Scale.Log ? 'SCALE_LOG' : 'SCALE_LINEAR',
    ),
};

export const getExperimentFileTree: DetApi<
  Service.ExperimentIdParams,
  Api.V1GetModelDefTreeResponse,
  Api.V1FileNode[]
> = {
  name: 'getModelDefTree',
  postProcess: (response) => response.files || [],
  request: (params) => {
    return detApi.Experiments.getModelDefTree(params.experimentId);
  },
};

export const getExperimentFileFromTree: DetApi<
  Api.V1GetModelDefFileRequest,
  Api.V1GetModelDefFileResponse,
  string
> = {
  name: 'getModelDefFile',
  postProcess: (response) => response.file || '',
  request: (params) => {
    return detApi.Experiments.getModelDefFile(
      // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
      params.experimentId!,
      { path: params.path },
    );
  },
};

type TrialWorkloadFilterOption =
  | 'FILTER_OPTION_UNSPECIFIED'
  | 'FILTER_OPTION_CHECKPOINT'
  | 'FILTER_OPTION_CHECKPOINT_OR_VALIDATION'
  | 'FILTER_OPTION_VALIDATION';

export const WorkloadFilterParamMap: Record<string, TrialWorkloadFilterOption> = {
  [Type.TrialWorkloadFilter.All]: 'FILTER_OPTION_UNSPECIFIED',
  [Type.TrialWorkloadFilter.Checkpoint]: 'FILTER_OPTION_CHECKPOINT',
  [Type.TrialWorkloadFilter.CheckpointOrValidation]: 'FILTER_OPTION_CHECKPOINT_OR_VALIDATION',
  [Type.TrialWorkloadFilter.Validation]: 'FILTER_OPTION_VALIDATION',
};

export const getTrialWorkloads: DetApi<
  Service.TrialWorkloadsParams,
  Api.V1GetTrialWorkloadsResponse,
  Type.TrialWorkloads
> = {
  name: 'getTrialWorkloads',
  postProcess: (response: Api.V1GetTrialWorkloadsResponse) => {
    return decoder.decodeTrialWorkloads(response);
  },
  request: (params: Service.TrialWorkloadsParams) =>
    detApi.Internal.getTrialWorkloads(
      params.id,
      params.orderBy,
      params.offset,
      params.limit,
      params.sortKey || 'batches',
      WorkloadFilterParamMap[params.filter || 'FILTER_OPTION_UNSPECIFIED'] ||
        'FILTER_OPTION_UNSPECIFIED',
      undefined,
      params.metricType
        ? params.metricType === Type.MetricType.Training
          ? 'METRIC_TYPE_TRAINING'
          : 'METRIC_TYPE_VALIDATION'
        : undefined,
    ),
};

/* Tasks */

export const getTask: DetApi<
  Service.GetTaskParams,
  Api.V1GetTaskResponse,
  Type.TaskItem | undefined
> = {
  name: 'getTask',
  postProcess: (response) => {
    return response.task ? decoder.mapV1Task(response.task) : undefined;
  },
  request: (params: Service.GetTaskParams) => detApi.Tasks.getTask(params.taskId),
};

export const getActiveTasks: DetApi<
  Record<string, never>,
  Api.V1GetActiveTasksCountResponse,
  Type.TaskCounts
> = {
  name: 'getActiveTasksCount',
  postProcess: (response) => response,
  request: (_, options) => detApi.Tasks.getActiveTasksCount(options),
};

/* Webhooks */

export const createWebhook: DetApi<Api.V1Webhook, Api.V1PostWebhookResponse, Type.Webhook> = {
  name: 'createWebhook',
  postProcess: (response) => decoder.mapV1Webhook(response.webhook),
  request: (params, options) => detApi.Webhooks.postWebhook(params, options),
};

export const deleteWebhook: DetApi<Service.GetWebhookParams, Api.V1DeleteWebhookResponse, void> = {
  name: 'deleteWebhook',
  postProcess: noOp,
  request: (params: Service.GetWebhookParams) => detApi.Webhooks.deleteWebhook(params.id),
};

export const getWebhooks: DetApi<EmptyParams, Api.V1GetWebhooksResponse, Type.Webhook[]> = {
  name: 'getWebhooks',
  postProcess: (response) => {
    return response.webhooks.map((hook) => decoder.mapV1Webhook(hook));
  },
  request: () => detApi.Webhooks.getWebhooks(),
};

export const testWebhook: DetApi<Service.GetWebhookParams, Api.V1TestWebhookResponse, void> = {
  name: 'testWebhook',
  postProcess: noOp,
  request: (params: Service.GetWebhookParams) => detApi.Webhooks.testWebhook(params.id),
};

/* Models */

export const getModels: DetApi<
  Service.GetModelsParams,
  Api.V1GetModelsResponse,
  Type.ModelPagination
> = {
  name: 'getModels',
  postProcess: (response) => {
    return {
      models: response.models.map((model) => decoder.mapV1Model(model)),
      pagination: response.pagination,
    };
  },
  request: (params: Service.GetModelsParams) =>
    detApi.Models.getModels(
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      params.name,
      params.description,
      params.labels,
      params.archived,
      undefined,
      undefined,
      getUserIds(params.users),
      undefined,
      params.workspaceIds,
    ),
};

export const getModel: DetApi<
  Service.GetModelParams,
  Api.V1GetModelResponse,
  Type.ModelItem | undefined
> = {
  name: 'getModel',
  postProcess: (response) => {
    return response.model ? decoder.mapV1Model(response.model) : undefined;
  },
  request: (params: Service.GetModelParams) => detApi.Models.getModel(params.modelName),
};

export const getModelDetails: DetApi<
  Service.GetModelDetailsParams,
  Api.V1GetModelVersionsResponse,
  Type.ModelVersions | undefined
> = {
  name: 'getModelDetails',
  postProcess: (response) => {
    return response.model != null && response.modelVersions != null
      ? decoder.mapV1ModelDetails(response)
      : undefined;
  },
  request: (params: Service.GetModelDetailsParams) =>
    detApi.Models.getModelVersions(
      params.modelName,
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
    ),
};

export const getModelVersion: DetApi<
  Service.GetModelVersionParams,
  Api.V1GetModelVersionResponse,
  Type.ModelVersion | undefined
> = {
  name: 'getModelVersion',
  postProcess: (response) => {
    return response.modelVersion ? decoder.mapV1ModelVersion(response.modelVersion) : undefined;
  },
  request: (params: Service.GetModelVersionParams) =>
    detApi.Models.getModelVersion(params.modelName, params.versionNum),
};

export const patchModel: DetApi<
  Service.PatchModelParams,
  Api.V1PatchModelResponse,
  Type.ModelItem | undefined
> = {
  name: 'patchModel',
  postProcess: (response) => (response.model ? decoder.mapV1Model(response.model) : undefined),
  request: (params: Service.PatchModelParams) =>
    detApi.Models.patchModel(params.modelName, params.body),
};

export const patchModelVersion: DetApi<
  Service.PatchModelVersionParams,
  Api.V1PatchModelVersionResponse,
  Type.ModelVersion | undefined
> = {
  name: 'patchModelVersion',
  postProcess: (response) =>
    response.modelVersion ? decoder.mapV1ModelVersion(response.modelVersion) : undefined,
  request: (params: Service.PatchModelVersionParams) =>
    detApi.Models.patchModelVersion(params.modelName, params.versionNum, params.body),
};

export const archiveModel: DetApi<Service.ArchiveModelParams, Api.V1ArchiveModelResponse, void> = {
  name: 'archiveModel',
  postProcess: noOp,
  request: (params: Service.GetModelParams) => detApi.Models.archiveModel(params.modelName),
};

export const unarchiveModel: DetApi<
  Service.ArchiveModelParams,
  Api.V1UnarchiveModelResponse,
  void
> = {
  name: 'unarchiveModel',
  postProcess: noOp,
  request: (params: Service.GetModelParams) => detApi.Models.unarchiveModel(params.modelName),
};

export const moveModel: DetApi<Service.MoveModelParams, Api.V1MoveModelResponse, void> = {
  name: 'moveModel',
  postProcess: noOp,
  request: (params: Service.MoveModelParams) =>
    detApi.Models.moveModel(params.modelName, {
      destinationWorkspaceId: params.destinationWorkspaceId,
      modelName: params.modelName,
    }),
};

export const deleteModel: DetApi<Service.DeleteModelParams, Api.V1DeleteModelResponse, void> = {
  name: 'deleteModel',
  postProcess: noOp,
  request: (params: Service.GetModelParams) => detApi.Models.deleteModel(params.modelName),
};

export const deleteModelVersion: DetApi<
  Service.DeleteModelVersionParams,
  Api.V1DeleteModelVersionResponse,
  void
> = {
  name: 'deleteModelVersion',
  postProcess: noOp,
  request: (params: Service.GetModelVersionParams) =>
    detApi.Models.deleteModelVersion(params.modelName, params.versionNum),
};

export const getModelLabels: DetApi<
  Service.GetWorkspaceModelsParams,
  Api.V1GetModelLabelsResponse,
  string[]
> = {
  name: 'getModelLabels',
  postProcess: (response) => response.labels || [],
  request: (params, options) => detApi.Models.getModelLabels(params.workspaceId, options),
};

export const postModel: DetApi<
  Service.PostModelParams,
  Api.V1PostModelResponse,
  Type.ModelItem | undefined
> = {
  name: 'postModel',
  postProcess: (response) => {
    return response.model ? decoder.mapV1Model(response.model) : undefined;
  },
  request: (params: Service.PostModelParams) =>
    detApi.Models.postModel({
      description: params.description,
      labels: params.labels,
      metadata: params.metadata,
      name: params.name,
      workspaceId: params.workspaceId,
    }),
};

export const postModelVersion: DetApi<
  Service.PostModelVersionParams,
  Api.V1PostModelVersionResponse,
  Type.ModelVersion | undefined
> = {
  name: 'postModelVersion',
  postProcess: (response) => {
    return response.modelVersion ? decoder.mapV1ModelVersion(response.modelVersion) : undefined;
  },
  request: (params: Service.PostModelVersionParams) =>
    detApi.Models.postModelVersion(params.modelName, params.body),
};

/* Workspaces */

export const getWorkspaces: DetApi<
  Service.GetWorkspacesParams,
  Api.V1GetWorkspacesResponse,
  Type.WorkspacePagination
> = {
  name: 'getWorkspaces',
  postProcess: (response) => {
    return {
      pagination: response.pagination,
      workspaces: response.workspaces.map(decoder.mapV1Workspace),
    };
  },
  request: (params, options) => {
    return detApi.Workspaces.getWorkspaces(
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      params.name,
      params.archived,
      undefined,
      getUserIds(params.users),
      params.pinned,
      options,
    );
  },
};

export const getWorkspace: DetApi<
  Service.GetWorkspaceParams,
  Api.V1GetWorkspaceResponse,
  Type.Workspace
> = {
  name: 'getWorkspace',
  postProcess: (response) => {
    return decoder.mapV1Workspace(response.workspace);
  },
  request: (params) => detApi.Workspaces.getWorkspace(params.id),
};

export const createWorkspace: DetApi<
  Api.V1PostWorkspaceRequest,
  Api.V1PostWorkspaceResponse,
  Type.Workspace
> = {
  name: 'createWorkspace',
  postProcess: (response) => {
    return decoder.mapV1Workspace(response.workspace);
  },
  request: (params, options) =>
    detApi.Workspaces.postWorkspace({ ...params, name: params.name }, options),
};

export const getWorkspaceMembers: DetApi<
  Service.GetWorkspaceMembersParams,
  Api.V1GetGroupsAndUsersAssignedToWorkspaceResponse,
  Type.WorkspaceMembersResponse
> = {
  name: 'getWorkspaceMembers',
  postProcess: (response) => ({
    assignments: response.assignments,
    groups: response.groups,
    usersAssignedDirectly: response.usersAssignedDirectly.map(decoder.mapV1User),
  }),
  request: (params) => {
    return detApi.RBAC.getGroupsAndUsersAssignedToWorkspace(params.workspaceId, params.nameFilter);
  },
};

export const getWorkspaceProjects: DetApi<
  Service.GetWorkspaceProjectsParams,
  Api.V1GetWorkspaceProjectsResponse,
  Type.ProjectPagination
> = {
  name: 'getWorkspaceProjects',
  postProcess: (response) => {
    return {
      pagination: response.pagination,
      projects: response.projects.map(decoder.mapV1Project),
    };
  },
  request: (params, options) => {
    return detApi.Workspaces.getWorkspaceProjects(
      params.id,
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      params.name,
      params.archived,
      undefined,
      getUserIds(params.users),
      options,
    );
  },
};

export const deleteWorkspace: DetApi<
  Service.DeleteWorkspaceParams,
  Api.V1DeleteWorkspaceResponse,
  Type.DeletionStatus
> = {
  name: 'deleteWorkspace',
  postProcess: decoder.mapDeletionStatus,
  request: (params) => detApi.Workspaces.deleteWorkspace(params.id),
};

export const patchWorkspace: DetApi<
  Service.PatchWorkspaceParams,
  Api.V1PatchWorkspaceResponse,
  Type.Workspace
> = {
  name: 'patchWorkspace',
  postProcess: (response) => {
    return decoder.mapV1Workspace(response.workspace);
  },
  request: (params, options) => {
    return detApi.Workspaces.patchWorkspace(
      params.id,
      { name: params.name?.trim(), ...params },
      options,
    );
  },
};

export const archiveWorkspace: DetApi<
  Service.ArchiveWorkspaceParams,
  Api.V1ArchiveWorkspaceResponse,
  void
> = {
  name: 'archiveWorkspace',
  postProcess: noOp,
  request: (params) => detApi.Workspaces.archiveWorkspace(params.id),
};

export const unarchiveWorkspace: DetApi<
  Service.UnarchiveWorkspaceParams,
  Api.V1UnarchiveWorkspaceResponse,
  void
> = {
  name: 'unarchiveWorkspace',
  postProcess: noOp,
  request: (params) => detApi.Workspaces.unarchiveWorkspace(params.id),
};

export const pinWorkspace: DetApi<Service.PinWorkspaceParams, Api.V1PinWorkspaceResponse, void> = {
  name: 'pinWorkspace',
  postProcess: noOp,
  request: (params) => detApi.Workspaces.pinWorkspace(params.id),
};

export const unpinWorkspace: DetApi<
  Service.UnpinWorkspaceParams,
  Api.V1UnpinWorkspaceResponse,
  void
> = {
  name: 'unpinWorkspace',
  postProcess: noOp,
  request: (params) => detApi.Workspaces.unpinWorkspace(params.id),
};

/* Projects */

export const getProject: DetApi<Service.GetProjectParams, Api.V1GetProjectResponse, Type.Project> =
  {
    name: 'getProject',
    postProcess: (response) => {
      return decoder.mapV1Project(response.project);
    },
    request: (params) => detApi.Projects.getProject(params.id),
  };

export const addProjectNote: DetApi<
  Service.AddProjectNoteParams,
  Api.V1AddProjectNoteResponse,
  Type.Note[]
> = {
  name: 'addProjectNote',
  postProcess: (response) => {
    return response.notes as Type.Note[];
  },
  request: (params) =>
    detApi.Projects.addProjectNote(params.id, {
      contents: params.contents,
      name: params.name,
    }),
};

export const setProjectNotes: DetApi<
  Service.SetProjectNotesParams,
  Api.V1PutProjectNotesResponse,
  Type.Note[]
> = {
  name: 'setProjectNotes',
  postProcess: (response) => {
    return response.notes as Type.Note[];
  },
  request: (params) =>
    detApi.Projects.putProjectNotes(params.projectId, {
      notes: params.notes,
      projectId: params.projectId,
    }),
};

export const createProject: DetApi<
  Api.V1PostProjectRequest,
  Api.V1PostProjectResponse,
  Type.Project
> = {
  name: 'createProject',
  postProcess: (response) => {
    return decoder.mapV1Project(response.project);
  },
  request: (params) =>
    detApi.Projects.postProject(params.workspaceId, {
      description: params.description?.trim(),
      name: params.name.trim(),
      workspaceId: params.workspaceId,
    }),
};

export const patchProject: DetApi<
  Service.PatchProjectParams,
  Api.V1PatchProjectResponse,
  Type.Project
> = {
  name: 'patchProject',
  postProcess: (response) => {
    return decoder.mapV1Project(response.project);
  },
  request: (params) =>
    detApi.Projects.patchProject(params.id, {
      description: params.description?.trim(),
      name: params.name?.trim(),
    }),
};

export const deleteProject: DetApi<
  Service.DeleteProjectParams,
  Api.V1DeleteProjectResponse,
  Type.DeletionStatus
> = {
  name: 'deleteProject',
  postProcess: decoder.mapDeletionStatus,
  request: (params) => detApi.Projects.deleteProject(params.id),
};

export const moveProject: DetApi<Api.V1MoveProjectRequest, Api.V1MoveProjectResponse, void> = {
  name: 'moveProject',
  postProcess: noOp,
  request: (params) =>
    detApi.Projects.moveProject(params.projectId, {
      destinationWorkspaceId: params.destinationWorkspaceId,
      projectId: params.projectId,
    }),
};

export const archiveProject: DetApi<
  Service.ArchiveProjectParams,
  Api.V1ArchiveProjectResponse,
  void
> = {
  name: 'archiveProject',
  postProcess: noOp,
  request: (params) => detApi.Projects.archiveProject(params.id),
};

export const unarchiveProject: DetApi<
  Service.UnarchiveProjectParams,
  Api.V1UnarchiveProjectResponse,
  void
> = {
  name: 'unarchiveProject',
  postProcess: noOp,
  request: (params) => detApi.Projects.unarchiveProject(params.id),
};

export const getProjectsByUserActivity: DetApi<
  Service.GetProjectsByUserActivityParams,
  Api.V1GetProjectsByUserActivityResponse,
  Type.Project[]
> = {
  name: 'getProjectsByUserActivity',
  postProcess: (response) => {
    return (response.projects || []).map((project) => decoder.mapV1Project(project));
  },
  request: (params: Service.GetProjectsByUserActivityParams) =>
    detApi.Projects.getProjectsByUserActivity(params.limit),
};

export const getProjectColumns: DetApi<
  Service.GetProjectColumnsParams,
  Api.V1GetProjectColumnsResponse,
  Api.V1ProjectColumn[]
> = {
  name: 'getProjectColumns',
  postProcess: (response) => response.columns,
  request: (params) => detApi.Internal.getProjectColumns(params.projectId),
};

/* Tasks */

const TASK_LIMIT = 1000;

export const getCommands: DetApi<
  Service.GetCommandsParams,
  Api.V1GetCommandsResponse,
  Type.CommandTask[]
> = {
  name: 'getCommands',
  postProcess: (response) =>
    (response.commands || []).map((command) => decoder.mapV1Command(command)),
  request: (params: Service.GetCommandsParams) =>
    detApi.Commands.getCommands(
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit ?? TASK_LIMIT,
      undefined,
      getUserIds(params.users),
      params.workspaceId,
    ),
};

export const getJupyterLabs: DetApi<
  Service.GetJupyterLabsParams,
  Api.V1GetNotebooksResponse,
  Type.CommandTask[]
> = {
  name: 'getJupyterLabs',
  postProcess: (response) =>
    (response.notebooks || []).map((jupyterLab) => decoder.mapV1Notebook(jupyterLab)),
  request: (params: Service.GetJupyterLabsParams) =>
    detApi.Notebooks.getNotebooks(
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit ?? TASK_LIMIT,
      undefined,
      getUserIds(params.users),
      params.workspaceId,
    ),
};

export const getShells: DetApi<
  Service.GetShellsParams,
  Api.V1GetShellsResponse,
  Type.CommandTask[]
> = {
  name: 'getShells',
  postProcess: (response) => (response.shells || []).map((shell) => decoder.mapV1Shell(shell)),
  request: (params: Service.GetShellsParams) =>
    detApi.Shells.getShells(
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit ?? TASK_LIMIT,
      undefined,
      getUserIds(params.users),
      params.workspaceId,
    ),
};

export const getTensorBoards: DetApi<
  Service.GetTensorBoardsParams,
  Api.V1GetTensorboardsResponse,
  Type.CommandTask[]
> = {
  name: 'getTensorBoards',
  postProcess: (response) =>
    (response.tensorboards || []).map((tensorboard) => decoder.mapV1TensorBoard(tensorboard)),
  request: (params: Service.GetTensorBoardsParams) =>
    detApi.TensorBoards.getTensorboards(
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit ?? TASK_LIMIT,
      undefined,
      getUserIds(params.users),
      params.workspaceId,
    ),
};

export const killCommand: DetApi<Service.CommandIdParams, Api.V1KillCommandResponse, void> = {
  name: 'killCommand',
  postProcess: noOp,
  request: (params: Service.CommandIdParams) => detApi.Commands.killCommand(params.commandId),
};

export const killJupyterLab: DetApi<Service.CommandIdParams, Api.V1KillNotebookResponse, void> = {
  name: 'killJupyterLab',
  postProcess: noOp,
  request: (params: Service.CommandIdParams) => detApi.Notebooks.killNotebook(params.commandId),
};

export const killShell: DetApi<Service.CommandIdParams, Api.V1KillShellResponse, void> = {
  name: 'killShell',
  postProcess: noOp,
  request: (params: Service.CommandIdParams) => detApi.Shells.killShell(params.commandId),
};

export const killTensorBoard: DetApi<Service.CommandIdParams, Api.V1KillTensorboardResponse, void> =
  {
    name: 'killTensorBoard',
    postProcess: noOp,
    request: (params: Service.CommandIdParams) =>
      detApi.TensorBoards.killTensorboard(params.commandId),
  };

export const getTemplates: DetApi<
  Service.GetTemplatesParams,
  Api.V1GetTemplatesResponse,
  Type.Template[]
> = {
  name: 'getTemplates',
  postProcess: (response) =>
    (response.templates || []).map((template) => decoder.mapV1Template(template)),
  request: (params: Service.GetTemplatesParams) =>
    detApi.Templates.getTemplates(
      params.sortBy,
      params.orderBy,
      params.offset,
      params.limit,
      params.name,
    ),
};

export const launchJupyterLab: DetApi<
  Service.LaunchJupyterLabParams,
  Api.V1LaunchNotebookResponse,
  Type.CommandResponse
> = {
  name: 'launchJupyterLab',
  postProcess: (response) => {
    return {
      command: decoder.mapV1Notebook(response.notebook),
      warnings: response.warnings || [],
    };
  },
  request: (params: Service.LaunchJupyterLabParams) => detApi.Notebooks.launchNotebook(params),
};

export const previewJupyterLab: DetApi<
  Service.LaunchJupyterLabParams,
  Api.V1LaunchNotebookResponse,
  RawJson
> = {
  name: 'previewJupyterLab',
  postProcess: (response) => response.config,
  request: (params: Service.LaunchJupyterLabParams) => detApi.Notebooks.launchNotebook(params),
};

export const launchTensorBoard: DetApi<
  Service.LaunchTensorBoardParams,
  Api.V1LaunchTensorboardResponse,
  Type.CommandResponse
> = {
  name: 'launchTensorBoard',
  postProcess: (response) => {
    return {
      command: decoder.mapV1TensorBoard(response.tensorboard),
      warnings: response.warnings || [],
    };
  },
  request: (params: Service.LaunchTensorBoardParams) =>
    detApi.TensorBoards.launchTensorboard(params),
};

/* Jobs */

export const getJobQueue: DetApi<
  Service.GetJobQParams,
  Api.V1GetJobsResponse,
  Service.GetJobsResponse
> = {
  name: 'getJobQ',
  postProcess: (response) => {
    response.jobs = response.jobs.filter((job) => !!job.summary);
    // we don't work with jobs without a summary in the ui yet
    return response as Service.GetJobsResponse;
  },
  request: (params: Service.GetJobQParams) =>
    detApi.Internal.getJobs(
      params.offset,
      params.limit,
      params.resourcePool,
      params.orderBy,
      decoder.decodeJobStates(params.states),
    ),
};

export const getJobQueueStats: DetApi<
  Service.GetJobQStatsParams,
  Api.V1GetJobQueueStatsResponse,
  Api.V1GetJobQueueStatsResponse
> = {
  name: 'getJobQStats',
  postProcess: identity,
  request: ({ resourcePools }) => detApi.Internal.getJobQueueStats(resourcePools),
};

export const updateJobQueue: DetApi<
  Api.V1UpdateJobQueueRequest,
  Api.V1UpdateJobQueueResponse,
  Api.V1UpdateJobQueueResponse
> = {
  name: 'updateJobQueue',
  postProcess: identity,
  request: (params: Api.V1UpdateJobQueueRequest) => detApi.Internal.updateJobQueue(params),
};
