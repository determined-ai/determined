import { AxiosResponse, CancelToken, CancelTokenSource, Method } from 'axios';

import { CommandType, TBSource } from 'types';

export interface ApiCommonParams {
  cancelToken?: CancelToken,
}

export interface HttpOptions {
  url: string;
  method?: Method;
  headers?: Record<string, unknown>;
  body?: Record<keyof unknown, unknown> | string;
}

interface ApiBase {
  name: string;
  stubbedResponse?: unknown;
  unAuthenticated?: boolean;
  // middlewares?: Middleware[]; // success/failure middlewares
}

// Designed for use with Swagger generated api bindings.
export interface DetApi<Input, DetOutput, Output> extends ApiBase {
  request: (params: Input) => Promise<DetOutput>;
  postProcess: (response: DetOutput) => Output;
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

export interface ExperimentsParams {
  states?: string[];
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

export interface ForkExperimentParams {
  parentId: number;
  experimentConfig: string;
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
  tail?: number;
  greaterThanId?: number;
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
