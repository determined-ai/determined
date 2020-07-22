import { CommandType, RunState, TBSourceType } from 'types';

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

export interface LaunchTensorboardParams {
  // currently we don't support launching from a mix of both trial ids and experiment ids
  ids: number[];
  type: TBSourceType;
}

export interface LogsParams {
  tail?: number;
  greaterThanId?: number;
}

export interface TrialLogsParams extends LogsParams {
  trialId: number;
}

export interface CommandLogsParams extends LogsParams {
  commandId: string;
  commandType: CommandType;
}

/* eslint-disable-next-line @typescript-eslint/ban-types */
export type EmptyParams = {}
