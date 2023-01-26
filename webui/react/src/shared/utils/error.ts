import { ValueOf } from 'shared/types';
import rootLogger from 'shared/utils/Logger';

import { isObject, isString } from './data';
import { LoggerInterface } from './Logger';

export const ERROR_NAMESPACE = 'EH';
export const DEFAULT_ERROR_MESSAGE = 'Unknown error encountered.';
const DEFAULT_LOGGER = rootLogger.extend(ERROR_NAMESPACE);

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

const defaultErrOptions: DetErrorOptions = {
  isUserTriggered: false,
  level: ErrorLevel.Error,
  logger: DEFAULT_LOGGER,
  silent: false,
  type: ErrorType.Unknown,
};

export const isError = (error: unknown): error is Error => {
  return error instanceof Error;
};

export const isDetError = (error: unknown): error is DetError => {
  return error instanceof DetError;
};

/**
 * used to preserve the public message potentially provided by lower levels where the error
 * was generated or rethrowed.
 *
 * @param e internal DetError object.
 * @param publicMessage a description of the error at this level.
 * @returns wrapped publicMessage if there was any provided at lower levels.
 */
export const wrapPublicMessage = (e: DetError | unknown, publicMessage: string): string => {
  if (!isDetError(e) || !e.publicMessage) return publicMessage;
  return `${publicMessage}: ${e.publicMessage}`;
};

/**
 * An expected Error with supplemental information on how it should be handled.
 */
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
