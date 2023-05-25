import { DataNode } from 'antd/lib/tree';
import * as t from 'io-ts';

import { Loadable } from 'utils/loadable';

export type Primitive = boolean | number | string;
export type RecordKey = string | number | symbol;
export type NullOrUndefined<T = undefined> = T | null | undefined;
export type Range<T = Primitive> = [T, T];

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

/**
 * DarkLight is a resolved form of `Mode` where we figure out
 * what `Mode.System` should ultimate resolve to (`Dark` vs `Light).
 */
export const DarkLight = {
  Dark: 'dark',
  Light: 'light',
} as const;

export type DarkLight = ValueOf<typeof DarkLight>;
export interface ClassNameProp {
  /** classname to be applied to the base element */
  className?: string;
}

export const ErrorLevel = {
  Error: 'error',
  Fatal: 'fatal',
  Warn: 'warning',
} as const;

export type ErrorLevel = ValueOf<typeof ErrorLevel>;

export const ErrorType = {
  // unexpected response structure.
  Api: 'api',

  // the issue is caused by unexpected/invalid user input.
  ApiBadResponse: 'apiBadResponse',

  // third-party api
  Assert: 'assert',

  // internal apis and server errors.
  Auth: 'auth',
  Input: 'input',
  Server: 'server',
  Ui: 'ui',
  Unknown: 'unknown', // assertion failure.
} as const;

export type ErrorType = ValueOf<typeof ErrorType>;

export type AnyMouseEvent = MouseEvent | React.MouseEvent;
export type AnyMouseEventHandler = (event: AnyMouseEvent) => void;

export type ErrorHandler = (e: unknown, options?: object) => void;

export interface TreeNode extends DataNode {
  /**
   * DataNode is the interface antd works with. DateNode properties we are interested in:
   *
   * key: we use V1FileNode.path
   * title: name of node
   * icon: custom Icon component
   */
  children?: TreeNode[];
  content: Loadable<string>;
  download?: string;
  get?: (path: string) => Promise<string>;
  isConfig?: boolean;
  isLeaf?: boolean;
}

export const MetricType = {
  Training: 'training',
  Validation: 'validation',
} as const;

export type MetricType = ValueOf<typeof MetricType>;

export interface Note {
  contents: string;
  name: string;
}

export interface User {
  displayName?: string;
  id: number;
  modifiedAt?: number;
  username: string;
}

export const LogLevel = {
  Critical: 'critical',
  Debug: 'debug',
  Error: 'error',
  Info: 'info',
  None: 'none',
  Trace: 'trace',
  Warning: 'warning',
} as const;

export type LogLevel = ValueOf<typeof LogLevel>;

// Disable `sort-keys` to sort LogLevel by higher severity levels
export const LogLevelFromApi = {
  Critical: 'LOG_LEVEL_CRITICAL',
  Error: 'LOG_LEVEL_ERROR',
  Warning: 'LOG_LEVEL_WARNING',
  // eslint-disable-next-line sort-keys-fix/sort-keys-fix
  Info: 'LOG_LEVEL_INFO',
  // eslint-disable-next-line sort-keys-fix/sort-keys-fix
  Debug: 'LOG_LEVEL_DEBUG',
  Trace: 'LOG_LEVEL_TRACE',
  Unspecified: 'LOG_LEVEL_UNSPECIFIED',
} as const;

export type LogLevelFromApi = ValueOf<typeof LogLevelFromApi>;

export interface Log {
  id: number | string;
  level?: LogLevel;
  message: string;
  meta?: string;
  time: string;
}
