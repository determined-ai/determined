import { AxiosResponse, Method } from 'axios';

import { CommandType, RunState, TBSourceType } from 'types';

export interface HttpOptions {
  url?: string;
  method?: Method;
  headers?: Record<string, unknown>;
  body?: Record<keyof unknown, unknown> | string;
}

export interface HttpApi<Input, Output>{
  name: string;
  httpOptions: (params: Input) => HttpOptions;
  postProcess?: (response: AxiosResponse<unknown>) => Output; // io type decoder.
  stubbedResponse?: unknown;
  // middlewares?: Middleware[]; // success/failure middlewares
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

// TODO in the following types the default id should probably be just "id"

export interface KillExpParams {
  experimentId: number;
}

export interface ForkExperimentParams {
  parentId: number;
  experimentConfig: string;
}

export interface KillCommandParams {
  commandId: string;
  commandType: CommandType;
}

export interface PatchExperimentParams {
  experimentId: number;
  body: Record<keyof unknown, unknown> | string;
}

export interface PatchExperimentState {
  experimentId: number;
  state: RunState;
}

export interface CreateNotebookParams {
  slots: number;
}

export interface CreateTensorboardParams {
  // currently we don't support launching from a mix of both trial ids and experiment ids
  ids: number[];
  type: TBSourceType;
}

export interface LogsParams {
  tail?: number;
  greaterThanId?: number;
}

export interface TaskLogsParams extends LogsParams {
  taskId: string;
  taskType: CommandType;
}

export interface TrialLogsParams extends LogsParams {
  trialId: number;
}

/* eslint-disable-next-line @typescript-eslint/ban-types */
export type EmptyParams = {}
