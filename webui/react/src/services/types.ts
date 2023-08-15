import { Dayjs } from 'dayjs';

import { FetchOptions, RecordKey, SingleEntityParams } from 'types';
import {
  DetailedUser,
  Job,
  Metadata,
  Metric,
  MetricType,
  Note,
  RunState,
  TrialWorkloadFilter,
} from 'types';

import * as Api from './api-ts-sdk/api';
export interface LoginResponse {
  token: string;
  user: DetailedUser;
}

export interface ApiSorter<T = string> {
  descend: boolean;
  key: T;
}

export type ExperimentDetailsParams = SingleEntityParams;
export type TrialDetailsParams = SingleEntityParams;

export interface TrialSummaryBaseParams {
  endBatches?: number;
  maxDatapoints: number;
  metrics: Metric[];
  metricType?: MetricType;
  startBatches?: number;
}

export interface TrialSummaryParams extends TrialSummaryBaseParams {
  trialId: number;
}

export interface TimeSeriesParams extends TrialSummaryBaseParams {
  trialIds: number[];
}

export interface TrialWorkloadsParams extends TrialDetailsParams, PaginationParams {
  filter: TrialWorkloadFilter;
  metricType?: MetricType | undefined;
  sortKey?: string;
}

export interface CommandIdParams {
  commandId: string;
}

export interface ExperimentIdParams {
  experimentId: number;
}

interface PaginationParams {
  limit?: number;
  offset?: number;
  orderBy?: 'ORDER_BY_UNSPECIFIED' | 'ORDER_BY_ASC' | 'ORDER_BY_DESC';
}

export interface GetTemplatesParams extends PaginationParams {
  name?: string;
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_NAME';
}

export interface GetExperimentsParams extends PaginationParams {
  archived?: boolean;
  description?: string;
  experimentIdFilter?: {
    incl?: Array<number>;
    notIn?: Array<number>;
  };
  labels?: Array<string>;
  name?: string;
  options?: never;
  projectId?: number;
  sortBy?:
    | 'SORT_BY_UNSPECIFIED'
    | 'SORT_BY_ID'
    | 'SORT_BY_DESCRIPTION'
    | 'SORT_BY_START_TIME'
    | 'SORT_BY_END_TIME'
    | 'SORT_BY_STATE'
    | 'SORT_BY_NUM_TRIALS'
    | 'SORT_BY_PROGRESS'
    | 'SORT_BY_USER'
    | 'SORT_BY_NAME';
  states?: Array<`STATE_${RunState}`>;
  userIds?: Array<number>;
  users?: Array<string>;
}

export interface SearchExperimentsParams extends PaginationParams {
  filter?: string;
  projectId?: number;
  sort?: string;
}

export interface BulkActionParams {
  excludedExperimentIds?: number[];
  experimentIds: number[];
  filters?: Api.V1BulkExperimentFilters;
}

export interface GetExperimentParams {
  id: number;
}

export interface getExperimentCheckpointsParams extends PaginationParams {
  id: number;
  sortBy?:
    | 'SORT_BY_UNSPECIFIED'
    | 'SORT_BY_UUID'
    | 'SORT_BY_TRIAL_ID'
    | 'SORT_BY_BATCH_NUMBER'
    | 'SORT_BY_END_TIME'
    | 'SORT_BY_STATE'
    | 'SORT_BY_SEARCHER_METRIC';
  states?: Array<
    'STATE_UNSPECIFIED' | 'STATE_ACTIVE' | 'STATE_COMPLETED' | 'STATE_ERROR' | 'STATE_DELETED'
  >;
}

export interface ExperimentLabelsParams {
  project_id?: number;
}

export interface GetTrialsParams extends PaginationParams, SingleEntityParams {
  sortBy?:
    | 'SORT_BY_UNSPECIFIED'
    | 'SORT_BY_ID'
    | 'SORT_BY_START_TIME'
    | 'SORT_BY_END_TIME'
    | 'SORT_BY_STATE';
  states?: Array<
    | 'STATE_UNSPECIFIED'
    | 'STATE_ACTIVE'
    | 'STATE_PAUSED'
    | 'STATE_STOPPING_COMPLETED'
    | 'STATE_STOPPING_CANCELED'
    | 'STATE_STOPPING_ERROR'
    | 'STATE_COMPLETED'
    | 'STATE_CANCELED'
    | 'STATE_ERROR'
    | 'STATE_DELETED'
  >;
}

export interface GetTaskParams {
  taskId: string;
}

export interface GetExperimentFileFromTreeParams {
  experimentId: number;
  filePath: string;
}

export interface GetModelsParams extends PaginationParams {
  archived?: boolean;
  description?: string;
  labels?: string[];
  name?: string;
  sortBy?:
    | 'SORT_BY_UNSPECIFIED'
    | 'SORT_BY_NAME'
    | 'SORT_BY_DESCRIPTION'
    | 'SORT_BY_CREATION_TIME'
    | 'SORT_BY_LAST_UPDATED_TIME'
    | 'SORT_BY_NUM_VERSIONS';
  users?: string[];
  workspaceIds?: number[];
}

export interface MoveModelParams {
  destinationWorkspaceId: number;
  modelName: string;
}

export interface GetModelParams {
  modelName: string;
}

export type ArchiveModelParams = GetModelParams;

export type DeleteModelParams = GetModelParams;

export interface GetModelDetailsParams extends PaginationParams {
  modelName: string;
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_VERSION' | 'SORT_BY_CREATION_TIME';
}

export interface GetModelVersionParams {
  modelName: string;
  versionNum: number;
}

export type DeleteModelVersionParams = GetModelVersionParams;

export interface PatchModelParams {
  body: {
    description?: string;
    labels?: string[];
    metadata?: Record<RecordKey, string | object>;
    name: string;
    notes?: string;
  };
  modelName: string;
}

export interface PatchModelVersionParams {
  body: {
    comment?: string;
    labels?: string[];
    metadata?: Record<RecordKey, string | object>;
    modelName: string;
    name?: string;
    notes?: string;
  };
  modelName: string;
  versionNum: number;
}

export interface PostModelParams {
  description?: string;
  labels?: string[];
  metadata?: Metadata;
  name: string;
  workspaceId?: number;
}

export interface PostModelVersionParams {
  body: {
    checkpointUuid: string;
    comment?: string;
    labels?: string[];
    metadata?: Metadata;
    modelName: string;
    name?: string;
    notes?: string;
  };
  modelName: string;
}

export interface CreateExperimentParams {
  activate?: boolean;
  experimentConfig: string;
  parentId: number;
  projectId: number;
}

export interface PatchExperimentParams extends ExperimentIdParams {
  body: Partial<{
    description: string;
    labels: string[];
    name: string;
    notes: string;
  }>;
}

export interface LaunchTensorBoardParams {
  excludedExperimentIds?: Array<number>;
  experimentIds?: Array<number>;
  trialIds?: Array<number>;
  workspaceId?: number;
  filters?: Api.V1BulkExperimentFilters;
}

export interface LaunchJupyterLabParams {
  config?: {
    description?: string;
    resources?: {
      resource_pool?: string;
      slots?: number;
    };
  };
  preview?: boolean;
  templateName?: string;
  workspaceId?: number;
}

export interface GetResourcePoolsParams extends PaginationParams {
  unbound?: boolean;
}

export interface GetCommandsParams extends FetchOptions, PaginationParams {
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_DESCRIPTION' | 'SORT_BY_START_TIME';
  users?: string[];
  workspaceId?: number;
}

export interface GetJupyterLabsParams extends FetchOptions, PaginationParams {
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_DESCRIPTION' | 'SORT_BY_START_TIME';
  users?: string[];
  workspaceId?: number;
}

export interface GetShellsParams extends FetchOptions, PaginationParams {
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_DESCRIPTION' | 'SORT_BY_START_TIME';
  users?: string[];
  workspaceId?: number;
}

export interface GetTensorBoardsParams extends FetchOptions, PaginationParams {
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_DESCRIPTION' | 'SORT_BY_START_TIME';
  users?: string[];
  workspaceId?: number;
}
export interface GetResourceAllocationAggregatedParams {
  endDate: Dayjs;
  period:
    | 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY'
    | 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY';
  startDate: Dayjs;
}

export interface GetJobQParams extends PaginationParams, FetchOptions {
  resourcePool: string;
  states?: Api.Jobv1State[];
}

export interface GetJobsResponse {
  pagination: Api.V1Pagination;
  jobs: Job[];
}
export interface GetJobQStatsParams extends FetchOptions {
  resourcePools?: string[];
}

export interface GetUsersParams extends PaginationParams {
  name?: string;
  sortBy?:
    | 'SORT_BY_UNSPECIFIED'
    | 'SORT_BY_USER_NAME'
    | 'SORT_BY_DISPLAY_NAME'
    | 'SORT_BY_ADMIN'
    | 'SORT_BY_ACTIVE'
    | 'SORT_BY_MODIFIED_TIME';
}
export interface GetUserParams {
  userId: number;
}
export interface PostUserParams {
  admin: boolean;
  displayName?: string;
  username: string;
}

export interface SetUserPasswordParams {
  password: string;
  userId: number;
}

export interface PatchUserParams {
  userId: number;
  userParams: Api.V1PatchUser;
}

export interface CreateGroupsParams {
  addUsers?: Array<number>;
  name: string;
}
export interface UpdateUserSettingParams {
  settings: Api.V1UserWebSetting | Api.V1UserWebSetting[];
}

export interface UpdateGroupParams {
  addUsers?: Array<number>;
  groupId: number;
  name?: string;
  removeUsers?: Array<number>;
}

export interface DeleteGroupParams {
  groupId: number;
}

export interface GetGroupParams {
  groupId: number;
}
export interface RemoveRolesFromGroupParams {
  groupId: number;
  roleIds: number[];
  scopeWorkspaceId?: number;
}

export interface AssignRolesToUserParams {
  roleIds: number[];
  scopeWorkspaceId?: number;
  userId: number;
}

export interface RemoveRolesFromUserParams {
  roleIds: number[];
  scopeWorkspaceId?: number;
  userId: number;
}

export type GetGroupsParams = PaginationParams & {
  userId?: number;
};

export interface AssignRolesToGroupParams {
  groupId: number;
  roleIds: number[];
  scopeWorkspaceId?: number;
}

export interface ListRolesParams {
  limit?: number;
  offset?: number;
}

export interface GetProjectParams {
  id: number;
}

export interface GetProjectExperimentsParams extends GetExperimentsParams {
  id: number;
}

export interface AddProjectNoteParams {
  contents: string;
  id: number;
  name: string;
}

export interface SetProjectNotesParams {
  notes: Note[];
  projectId: number;
}

export interface GetWorkspacesParams extends PaginationParams {
  archived?: boolean;
  name?: string;
  pinned?: boolean;
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_NAME';
  users?: string[];
}

export interface GetWorkspaceProjectsParams extends PaginationParams {
  archived?: boolean;
  id: number;
  name?: string;
  sortBy?:
    | 'SORT_BY_UNSPECIFIED'
    | 'SORT_BY_CREATION_TIME'
    | 'SORT_BY_LAST_EXPERIMENT_START_TIME'
    | 'SORT_BY_NAME'
    | 'SORT_BY_DESCRIPTION';
  users?: string[];
}

export interface GetWorkspaceModelsParams {
  workspaceId?: number;
}

export interface GetWorkspaceMembersParams {
  nameFilter?: string;
  workspaceId: number;
}

export interface DeleteProjectParams {
  id: number;
}

export interface PatchWorkspaceParams extends Api.V1PatchWorkspace {
  id: number;
}

export interface PatchProjectParams extends Api.V1PatchProject {
  id: number;
}

export interface ArchiveProjectParams {
  id: number;
}

export type UnarchiveProjectParams = ArchiveProjectParams;

export interface GetProjectColumnsParams {
  id: number;
}

export interface GetProjectNumericMetricsRangeParams {
  id: number;
}

export interface ActionWorkspaceParams {
  workspaceId: number;
}

export interface GetWebhookParams {
  id: number;
}

export interface SearchRolesAssignableToScopeParams {
  limit?: number;
  offset?: number;
  workspaceId: number;
}

export interface GetProjectsByUserActivityParams {
  limit?: number;
}

export interface GetResourcePoolBindingsParams {
  resourcePoolName: string;
}

export interface ModifyResourcePoolBindingsParams {
  resourcePoolName: string;
  workspaceIds: number[];
}
