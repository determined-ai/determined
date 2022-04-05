import { Dayjs } from 'dayjs';

import { DetailedUser, Job, Metadata, RecordKey } from 'types';

import * as Api from './api-ts-sdk/api';

export interface FetchOptions {
  signal?: AbortSignal;
}

interface ApiBase {
  name: string;
  stubbedResponse?: unknown;
  unAuthenticated?: boolean;
  // middlewares?: Middleware[]; // success/failure middlewares
}

// Designed for use with Swagger generated api bindings.
export interface DetApi<Input, DetOutput, Output> extends ApiBase {
  postProcess: (response: DetOutput) => Output;
  request: (params: Input, options?: FetchOptions) => Promise<DetOutput>;
  stubbedResponse?: DetOutput;
}

export interface ApiState<T> {
  data?: T;
  error?: Error;
  isLoading: boolean;
}

export interface LoginResponse {
  token: string;
  user: DetailedUser;
}

export interface ApiSorter<T = string> {
  descend: boolean;
  key: T;
}

export interface SingleEntityParams {
  id: number;
}

export type ExperimentDetailsParams = SingleEntityParams;
export type TrialDetailsParams = SingleEntityParams;

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
  labels?: Array<string>;
  name?: string;
  options?: never;
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_DESCRIPTION' | 'SORT_BY_START_TIME'
  | 'SORT_BY_END_TIME' | 'SORT_BY_STATE' | 'SORT_BY_NUM_TRIALS' | 'SORT_BY_PROGRESS'
  | 'SORT_BY_USER' | 'SORT_BY_NAME';
  states?: Array<'STATE_UNSPECIFIED' | 'STATE_ACTIVE' | 'STATE_PAUSED'
  | 'STATE_STOPPING_COMPLETED' | 'STATE_STOPPING_CANCELED' | 'STATE_STOPPING_ERROR'
  | 'STATE_COMPLETED' | 'STATE_CANCELED' | 'STATE_ERROR' | 'STATE_DELETED'>;
  users?: Array<string>;
}

export interface GetExperimentParams {
  id: number;
}

export interface GetTrialsParams extends PaginationParams, SingleEntityParams {
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_START_TIME'
  | 'SORT_BY_END_TIME' | 'SORT_BY_STATE';
  states?: Array<'STATE_UNSPECIFIED' | 'STATE_ACTIVE' | 'STATE_PAUSED'
  | 'STATE_STOPPING_COMPLETED' | 'STATE_STOPPING_CANCELED' | 'STATE_STOPPING_ERROR'
  | 'STATE_COMPLETED' | 'STATE_CANCELED' | 'STATE_ERROR' | 'STATE_DELETED'>;
}

export interface GetTaskParams {
  taskId: string;
}

export interface GetModelsParams extends PaginationParams {
  archived?: boolean;
  description?: string;
  labels?: string[];
  name?: string;
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_NAME' | 'SORT_BY_DESCRIPTION'
  | 'SORT_BY_CREATION_TIME' | 'SORT_BY_LAST_UPDATED_TIME' | 'SORT_BY_NUM_VERSIONS';
  users?: string[];
}

export interface GetModelParams {
  modelName: string;
}

export type ArchiveModelParams = GetModelParams;

export type DeleteModelParams = GetModelParams;

export interface GetModelDetailsParams extends PaginationParams {
  modelName: string;
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_VERSION' | 'SORT_BY_CREATION_TIME'
}

export interface GetModelVersionParams {
  modelName: string;
  versionId: number;
}

export type DeleteModelVersionParams = GetModelVersionParams;

export interface PatchModelParams {
  body: {
    description?: string;
    labels?: string[];
    metadata?: Record<RecordKey, string>;
    name: string;
    notes?: string;
  }
  modelName: string;
}

export interface PatchModelVersionParams {
  body: {
    comment?: string;
    labels?: string[];
    metadata?: Record<RecordKey, string>;
    modelName: string;
    name?: string;
    notes?: string;
  }
  modelName: string;
  versionId: number;
}

export interface PostModelParams {
  description?: string;
  labels?: string[];
  metadata?: Metadata;
  name: string;
  username?: string;
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
  }
  modelName: string;
}

export interface CreateExperimentParams {
  activate?: boolean;
  experimentConfig: string;
  parentId: number;
}

export interface PatchExperimentParams extends ExperimentIdParams {
  body: Partial<{
    description: string,
    labels: string[],
    name: string,
    notes: string;
  }>
}

export interface LaunchTensorBoardParams {
  experimentIds?: Array<number>;
  trialIds?: Array<number>;
}

export interface LaunchJupyterLabParams {
  config?: {
    description?: string;
    resources?: {
      resource_pool?: string;
      slots?: number;
    }
  };
  preview?: boolean;
  templateName?: string;
}

/* eslint-disable-next-line @typescript-eslint/ban-types */
export type EmptyParams = {}

export interface GetCommandsParams extends FetchOptions, PaginationParams {
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_DESCRIPTION' | 'SORT_BY_START_TIME';
}

export interface GetJupyterLabsParams extends FetchOptions, PaginationParams {
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_DESCRIPTION' | 'SORT_BY_START_TIME';
}

export interface GetShellsParams extends FetchOptions, PaginationParams {
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_DESCRIPTION' | 'SORT_BY_START_TIME';
}

export interface GetTensorBoardsParams extends FetchOptions, PaginationParams {
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_DESCRIPTION' | 'SORT_BY_START_TIME';
}

export interface GetResourceAllocationAggregatedParams {
  endDate: Dayjs,
  period: 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY'
  | 'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY',
  startDate: Dayjs,
}

export interface GetJobQParams extends PaginationParams, FetchOptions {
  resourcePool: string;
}

export interface GetJobsResponse extends Api.V1GetJobsResponse {
  jobs: Job[];
}
export interface GetJobQStatsParams extends FetchOptions {
  resourcePools?: string[];
}

export interface SetUserPasswordParams {
  password: string;
  username: string;
}

export interface PatchUserParams {
  userParams: {
    displayName: string;
  };
  username: string;
}
