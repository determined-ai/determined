import { AxiosResponse, CancelToken, CancelTokenSource, Method } from 'axios';

import { CommandType, TBSource } from 'types';

export interface ApiCommonParams {
  cancelToken?: CancelToken,
}

export interface FetchOptionParams {
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
  request: (params: Input) => Promise<DetOutput>;
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

export interface GetExperimentsParams {
  archived?: boolean;
  description?: string;
  labels?: Array<string>;
  limit?: number;
  offset?: number;
  options?: never;
  orderBy?: 'ORDER_BY_UNSPECIFIED' | 'ORDER_BY_ASC' | 'ORDER_BY_DESC';
  sortBy?: 'SORT_BY_UNSPECIFIED' | 'SORT_BY_ID' | 'SORT_BY_DESCRIPTION' | 'SORT_BY_START_TIME'
  | 'SORT_BY_END_TIME' | 'SORT_BY_STATE' | 'SORT_BY_NUM_TRIALS' | 'SORT_BY_PROGRESS'
  | 'SORT_BY_USER';
  states?: Array<'STATE_UNSPECIFIED' | 'STATE_ACTIVE' | 'STATE_PAUSED'
  | 'STATE_STOPPING_COMPLETED' | 'STATE_STOPPING_CANCELED' | 'STATE_STOPPING_ERROR'
  | 'STATE_COMPLETED' | 'STATE_CANCELED' | 'STATE_ERROR' | 'STATE_DELETED'>;
  users?: Array<string>;
}

export interface CreateExperimentParams {
  experimentConfig: string;
  parentId: number;
}

export interface PatchExperimentParams extends ExperimentIdParams {
  body: Partial<{
    description: string,
    labels: string[],
  }>
}

export interface CreateNotebookParams {
  slots: number;
}

export type CreateTensorboardParams = TBSource;

export interface LogsParams {
  greaterThanId?: number;
  tail?: number;
}

export interface TaskLogsParams extends LogsParams {
  taskId: string;
  taskType: CommandType;
}

export interface TrialLogsParams extends LogsParams {
  experimentId: number;
  trialId: number;
}

/* eslint-disable-next-line @typescript-eslint/ban-types */
export type EmptyParams = {}
