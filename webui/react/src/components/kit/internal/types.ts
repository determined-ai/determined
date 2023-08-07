import { DataNode } from 'antd/lib/tree';
import * as t from 'io-ts';

import { isObject, isString } from 'components/kit/internal/functions';
import rootLogger, { LoggerInterface } from 'components/kit/internal/Logger';

export type Primitive = boolean | number | string;
export type RecordKey = string | number | symbol;
export type NullOrUndefined<T = undefined> = T | null | undefined;
export type Range<T = Primitive> = [T, T];

export type ValueOf<T> = T[keyof T];

type Without<T, U> = { [P in Exclude<keyof T, keyof U>]?: never };
export type XOR<T, U> = T | U extends object ? (Without<T, U> & U) | (Without<U, T> & T) : T | U;

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

export const ERROR_NAMESPACE = 'EH';

export const isError = (error: unknown): error is Error => {
  return error instanceof Error;
};
const DEFAULT_LOGGER = rootLogger.extend(ERROR_NAMESPACE);

export const DEFAULT_ERROR_MESSAGE = 'Unknown error encountered.';
const defaultErrOptions: DetErrorOptions = {
  isUserTriggered: false,
  level: ErrorLevel.Error,
  logger: DEFAULT_LOGGER,
  silent: false,
  type: ErrorType.Unknown,
};

export interface DetErrorOptions {
  id?: string; // slug unique to each place in the codebase that we will use this.
  isUserTriggered?: boolean; // whether the error was triggered by an active interaction.
  level?: ErrorLevel;
  logger?: LoggerInterface;
  payload?: unknown;
  publicMessage?: string;
  publicSubject?: string;
  silent?: boolean;
  type?: ErrorType;
}

export class DetError extends Error implements DetErrorOptions {
  id?: string;
  isUserTriggered: boolean;
  level: ErrorLevel;
  logger: LoggerInterface; // CHECK: do we want this attached to DetError?
  payload?: unknown;
  publicMessage?: string;
  publicSubject?: string;
  silent: boolean;
  type: ErrorType;
  isHandled: boolean;
  /** the wrapped error if one was provided. */
  sourceErr: unknown;

  /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
  constructor(e?: any, options: DetErrorOptions = {}) {
    const defaultMessage = isError(e) ? e.message : isString(e) ? e : DEFAULT_ERROR_MESSAGE;
    const message = options.publicSubject || options.publicMessage || defaultMessage;
    super(message);

    const eOpts: Partial<DetErrorOptions> = {};
    if (isObject(e)) {
      if ('id' in e && e.id != null) eOpts.id = e.id;
      if ('isUserTriggered' in e && e.isUserTriggered != null)
        eOpts.isUserTriggered = e.isUserTriggered;
      if ('level' in e && e.level != null) eOpts.level = e.level;
      if ('logger' in e && e.logger != null) eOpts.logger = e.logger;
      if ('payload' in e && e.payload != null) eOpts.payload = e.payload;
      if ('publicMessage' in e && e.publicMessage != null) eOpts.publicMessage = e.publicMessage;
      if ('silent' in e && e.silent != null) eOpts.silent = e.silent;
      if ('type' in e && e.type != null) eOpts.type = e.type;
    }

    this.loadOptions({ ...defaultErrOptions, ...eOpts, ...options });
    this.isHandled = false;
    this.sourceErr = e;
  }

  loadOptions(options: DetErrorOptions): void {
    Object.assign(this, options);
  }
}

export type ErrorHandler = (
  error: DetError | unknown,
  options?: DetErrorOptions,
) => DetError | void;

export interface TreeNode extends DataNode {
  /**
   * DataNode is the interface antd works with. DateNode properties we are interested in:
   *
   * key: we use V1FileNode.path
   * title: name of node
   * icon: custom Icon component
   */
  children?: TreeNode[];
  download?: string;
  get?: (path: string) => Promise<string>;
  isLeaf?: boolean;
  subtitle?: string;
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
