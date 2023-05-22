import * as t from 'io-ts';

export type Primitive = boolean | number | string;
export type RecordKey = string | number | symbol;

export type ValueOf<T> = T[keyof T];

export const Scale = {
  Linear: 'linear',
  Log: 'log',
} as const;

export type Scale = ValueOf<typeof Scale>;

interface EndTimes {
  endTime?: string;
}

const CheckpointState = {
  Active: 'ACTIVE',
  Completed: 'COMPLETED',
  Deleted: 'DELETED',
  Error: 'ERROR',
  Unspecified: 'UNSPECIFIED',
} as const;
type CheckpointState = ValueOf<typeof CheckpointState>;

interface BaseWorkload extends EndTimes {
  totalBatches: number;
}
interface CheckpointWorkload extends BaseWorkload {
  resources?: Record<string, number>;
  state: CheckpointState;
  uuid?: string;
}
interface CheckpointWorkloadExtended extends CheckpointWorkload {
  experimentId: number;
  trialId: number;
}

export type XAxisVal = number;
export type CheckpointsDict = Record<XAxisVal, CheckpointWorkloadExtended>;

export interface SettingsConfigProp<A> {
  defaultValue: A;
  skipUrlEncoding?: boolean;
  storageKey: string;
  type: t.Type<A>;
}
export interface SettingsConfig<T> {
  settings: { [K in keyof T]: SettingsConfigProp<T[K]> };
  storagePath: string;
}

export interface FetchArgs {
  url: string;
  // eslint-disable-next-line
  options: any;
}
