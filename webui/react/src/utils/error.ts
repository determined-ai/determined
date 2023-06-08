import { ArgsProps, NotificationInstance } from 'antd/lib/notification/interface';

import { telemetryInstance } from 'hooks/useTelemetry';
import { paths } from 'routes/utils';
import { ValueOf } from 'types';
import { notification as antdNotification } from 'utils/dialogApi';
import { LoggerInterface } from 'utils/Logger';
import rootLogger from 'utils/Logger';
import { routeToReactUrl } from 'utils/routes';
import { isAborted, isAuthFailure } from 'utils/service';
import { listToStr } from 'utils/string';

import { isObject, isString } from './data';

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

const errorLevelMap = {
  [ErrorLevel.Error]: 'error',
  [ErrorLevel.Fatal]: 'error',
  [ErrorLevel.Warn]: 'warn',
};

const openNotification = (e: DetError) => {
  const key = errorLevelMap[e.level] as keyof NotificationInstance;
  const notification = antdNotification[key] as (args: ArgsProps) => void;

  notification?.({
    description: e.publicMessage || '',
    key: e.name,
    message: e.publicSubject || listToStr([e.type, e.level]),
  });
};

const log = (e: DetError) => {
  const key = errorLevelMap[e.level] as keyof LoggerInterface;
  const message = listToStr([`${e.type}:`, e.publicMessage, e.message]);
  e.logger[key](message);
  e.logger[key](e);
};

// Handle a warning to the user in the UI
export const handleWarning = (warningOptions: DetErrorOptions): void => {
  // Error object is null because this is just a warning
  const detWarning = new DetError(null, warningOptions);

  openNotification(detWarning);
};

// Handle an error at the point that you'd want to stop bubbling it up. Avoid handling
// and re-throwing.
const handleError = (error: DetError | unknown, options?: DetErrorOptions): DetError | void => {
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
      // eslint-disable-next-line no-console
      console.warn(`Error "${e.message}" is handled twice.`);
    }
    return;
  }
  e.isHandled = true;

  // Redirect to logout if Auth failure detected (auth token is no longer valid).`
  if (isAuthFailure(e)) {
    // This check accounts for requests that had not been aborted properly prior
    // to the page dismount and end up throwing after the user is logged out.
    const path = window.location.pathname;
    if (!path.includes(paths.login()) && !path.includes(paths.logout())) {
      routeToReactUrl(paths.logout());
    }
  }

  // TODO add support for checking, saving, and dismissing class of errors as a user preference
  // using id.
  const skipNotification = e.silent || (e.level === ErrorLevel.Warn && !e.publicMessage);
  if (!skipNotification) openNotification(e);

  // TODO generate stack trace if error is missing? http://www.stacktracejs.com/

  log(e);

  // TODO SEP handle transient failures? eg only take action IF.. (requires keeping state)

  // Report to segment.
  telemetryInstance.track(`${ERROR_NAMESPACE}: ${e.level}`, e);

  // TODO SEP capture a screenshot or more context (generate a call stack)?
  // https://stackblitz.com/edit/react-screen-capture?file=index.js

  return e;
};

export default handleError;
