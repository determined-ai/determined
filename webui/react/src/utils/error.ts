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

const ERROR_NAMESPACE = 'EH';
const DEFAULT_ERROR_MESSAGE = 'Unknown error encountered.';
const DEFAULT_LOGGER = new Logger(ERROR_NAMESPACE);

export const isError = (error: unknown): error is Error => {
  return error instanceof Error;
};

export const isDetError = (error: unknown): error is DetError => {
  return error instanceof DetError;
};

const defaultErrOptions: DetErrorOptions = {
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
    message: e.publicSubject || listToStr([ e.type, e.level ]),
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
  // Ignore request cancellation errors.
  if (isAborted(error)) return;

  let e: DetError | undefined;
  if (isDetError(error)) {
    e = error;
    if (options) e.loadOptions(options);
  } else {
    e = new DetError(error, options);
  }

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
  const skipNotification = e.silent || (e.level === ErrorLevel.Warn && !e.publicMessage);
  if (!skipNotification) openNotification(e);

  // TODO generate stack trace if error is missing? http://www.stacktracejs.com/

  // Log the error if needed.
  if (!e.silent) log(e);

  // TODO SEP handle transient failures? eg only take action IF.. (requires keeping state)

  // Report to segment.
  telemetryInstance.track(`${ERROR_NAMESPACE}: ${e.level}`, e);

  // TODO SEP capture a screenshot or more context (generate a call stack)?
  // https://stackblitz.com/edit/react-screen-capture?file=index.js
};

export default handleError;
