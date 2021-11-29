import { AxiosResponse, CancelToken, CancelTokenSource, Method } from 'axios';
import { Dayjs } from 'dayjs';

import { CommandType, DetailedUser, RecordKey } from 'types';

export interface ApiCommonParams {
  cancelToken?: CancelToken,
}

export interface FetchOptions {
  signal?: AbortSignal;
}

export interface HttpOptions {
  body?: Record<keyof unknown, unknown> | string;
  headers?: Record<string, unknown>;
  method?: Method;
  url: string;
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
export interface HttpApi<Input, Output> extends ApiBase {
  httpOptions: (params: Input) => HttpOptions;
  postProcess: (response: AxiosResponse<unknown>) => Output; // io type decoder.
}

export interface ApiState<T> {
  data?: T;
  error?: Error;
  isLoading: boolean;
  source?: CancelTokenSource;
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
  modelId: number;
}

export type ArchiveModelParams = GetModelParams;

export type DeleteModelParams = GetModelParams;

export interface GetModelDetailsParams extends PaginationParams {
  modelId: number;
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_VERSION' | 'SORT_BY_CREATION_TIME'
}

export interface GetModelVersionParams {
  modelId: number;
  versionId: number;
}

export type DeleteModelVersionParams = GetModelVersionParams;

export interface PatchModelParams {
  body: {
    description?: string;
    id: number;
    labels?: string[];
    metadata?: Record<RecordKey, string>;
    name?: string;
    notes?: string;
  }
  modelId: number;
}

export interface PatchModelVersionParams {
  body: {
    comment?: string;
    id: number;
    labels?: string[];
    metadata?: Record<RecordKey, string>;
    name?: string;
    notes?: string;
  }
  modelId: number;
  versionId: number;
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

export interface LogsParams {
  greaterThanId?: number;
  tail?: number;
}

export interface TaskLogsParams extends LogsParams {
  taskId: string;
  taskType: CommandType;
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
