import { CommandType } from 'types';

export interface ExperimentsParams {
  states?: string[];
}

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
