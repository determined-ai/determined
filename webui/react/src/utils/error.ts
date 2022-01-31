import { notification as antdNotification } from 'antd';
import { ArgsProps, NotificationApi } from 'antd/lib/notification';

import { telemetryInstance } from 'hooks/useTelemetry';
import history from 'routes/history';
import { filterOutLoginLocation, paths } from 'routes/utils';
import { isAborted } from 'services/utils';
import Logger, { LoggerInterface } from 'utils/Logger';
import { listToStr } from 'utils/string';

import { isString } from './data';

export interface DetErrorOptions {
  id?: string;                // slug unique to each place in the codebase that we will use this.
  isUserTriggered?: boolean;  // whether the error was triggered by an active interaction.
  level?: ErrorLevel;
  logger?: LoggerInterface;
  name?: string;
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

const ERROR_NAMESPACE = 'EH';
const DEFAULT_ERROR_MESSAGE = 'Unknown error encountered.';
const DEFAULT_LOGGER = new Logger(ERROR_NAMESPACE);

export const isError = (error: unknown): error is Error => {
  return error instanceof Error;
};

export const isDetError = (error: unknown): error is DetError => {
  return error instanceof DetError;
};

const defaultErrorOptions: DetErrorOptions = {
  isUserTriggered: false,
  level: ErrorLevel.Error,
  logger: DEFAULT_LOGGER,
  silent: false,
  type: ErrorType.Unknown,
};

// An expected Error with supplemental information on
// how it should be handled.
export class DetError extends Error implements DetErrorOptions {
  id?: string;
  isHandled: boolean;
  isUserTriggered: boolean;
  level: ErrorLevel;
  logger: LoggerInterface;
  original?: unknown;
  payload?: unknown;
  publicMessage?: string;
  publicSubject?: string;
  silent: boolean;
  type: ErrorType;

  constructor(e?: unknown, options: DetErrorOptions = {}) {
    const defaultMessage = isError(e) ? e.message : (isString(e) ? e : DEFAULT_ERROR_MESSAGE);
    const message = options.publicSubject || options.publicMessage || defaultMessage;
    super(message);

    // Maintains proper stack trace for where our error was thrown.
    if (Error.captureStackTrace) Error.captureStackTrace(this, DetError);

    // Override DetError defaults with options.
    Object.assign(this, { ...defaultErrorOptions, ...options });

    // Save original error being passed in.
    this.original = e;
    this.name = e instanceof Error ? e.name : 'Error';

    // Flag indicating whether this error has previously been handled by `handleError`.
    this.isHandled = false;
  }
}

const errorLevelMap = {
  [ErrorLevel.Error]: 'error',
  [ErrorLevel.Fatal]: 'error',
  [ErrorLevel.Warn]: 'warn',
};

const openNotification = (e: DetError) => {
  const key = errorLevelMap[e.level] as keyof NotificationApi;
  const notification = antdNotification[key] as (args: ArgsProps) => void;

  notification?.({
    description: e.publicMessage || '',
    message: e.publicSubject || e.message || listToStr([ e.type, e.level ]),
  });
};

const log = (e: DetError) => {
  const key = errorLevelMap[e.level] as keyof LoggerInterface;
  const message = listToStr([ `${e.type}:`, e.publicMessage, e.message ]);
  e.logger[key](message);
  e.logger[key](e);
};

// Handle an error at the point that you'd want to stop bubbling it up. Avoid handling
// and re-throwing.
const handleError = (error: DetError | unknown, options?: DetErrorOptions): void => {
  // Wrap existing error with more info via `options`.
  const e: DetError = new DetError(error, options);

  // Ignore request cancellation errors.
  if (isAborted(e)) return;

  // Ensure `handleError` doesn't handle the same exact error more than once.
  if (e.isHandled) {
    if (process.env.IS_DEV) {
      console.warn(`Error "${e.message}" is handled twice.`);
    }
    return;
  }
  e.isHandled = true;

  // Redirect to logout if Auth failure detected (auth token is no longer valid).`
  if (e.type === ErrorType.Auth) {
    history.push(paths.logout(), { loginRedirect: filterOutLoginLocation(window.location) });
  }

  // TODO add support for checking, saving, and dismissing class of errors as a user preference
  // using id.
  if (!e.silent) openNotification(e);

  // Log the error.
  log(e);

  // TODO generate stack trace if error is missing? http://www.stacktracejs.com/
  // TODO SEP handle transient failures? eg only take action IF.. (requires keeping state)

  // Report to segment.
  telemetryInstance.track(`${ERROR_NAMESPACE}: ${e.level}`, e);

  // TODO SEP capture a screenshot or more context (generate a call stack)?
  // https://stackblitz.com/edit/react-screen-capture?file=index.js
};

export default handleError;
