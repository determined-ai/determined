import rootLogger from 'shared/utils/Logger';

import { isString } from './data';
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

export enum ErrorLevel {
  Fatal = 'fatal',
  Error = 'error',
  Warn = 'warning',
}

export enum ErrorType {
  Server = 'server', // internal apis and server errors.
  Auth = 'auth',
  Unknown = 'unknown',
  Ui = 'ui',
  Input = 'input', // the issue is caused by unexpected/invalid user input.
  ApiBadResponse = 'apiBadResponse', // unexpected response structure.
  Api = 'api', // third-party api
}

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
  logger: LoggerInterface;
  payload?: unknown;
  publicMessage?: string;
  publicSubject?: string;
  silent: boolean;
  type: ErrorType;
  isHandled: boolean;

  constructor(e?: unknown, options: DetErrorOptions = {}) {
    const defaultMessage = isError(e) ? e.message : (isString(e) ? e : DEFAULT_ERROR_MESSAGE);
    const message = options.publicSubject || options.publicMessage || defaultMessage;
    super(message);

    const eOpts: DetErrorOptions = isDetError(e) ? {
      id: e.id,
      isUserTriggered: e.isUserTriggered,
      level: e.level,
      logger: e.logger,
      payload: e.payload,
      publicMessage: e.publicMessage,
      publicSubject: e.publicSubject,
      silent: e.silent,
      type: e.type,
    } : {};

    this.loadOptions({ ...defaultErrOptions, ...eOpts, ...options });
    this.isHandled = false;
  }

  loadOptions(options: DetErrorOptions): void {
    Object.assign(this, options);
  }
}
