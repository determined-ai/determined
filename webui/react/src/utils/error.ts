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
  isUserTriggered?: boolean; // whether the error was caused by an active interaction.
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
  Input = 'input',
  ApiBadResponse = 'apiBadResponse',
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

export class DetError extends Error {
  id?: string;
  isUserTriggered: boolean;
  level: ErrorLevel;
  logger: LoggerInterface;
  payload?: unknown;
  publicMessage?: string;
  publicSubject?: string;
  silent: boolean;
  type: ErrorType;

  constructor(e?: unknown, options: DetErrorOptions = {}) {
    const defaultMessage = isError(e) ? e.message : (isString(e) ? e : DEFAULT_ERROR_MESSAGE);
    const message = options.publicSubject || options.publicMessage || defaultMessage;
    super(message);

    const detError = isDetError(e) ? e : undefined;
    this.id = options.id || detError?.id || undefined;
    this.isUserTriggered = options.isUserTriggered || detError?.isUserTriggered || false;
    this.level = options.level || detError?.level || ErrorLevel.Error;
    this.logger = options.logger || detError?.logger || DEFAULT_LOGGER;
    this.payload = options.payload || detError?.payload || undefined;
    this.publicMessage = options.publicMessage || detError?.publicMessage || undefined;
    this.publicSubject = options.publicSubject || detError?.publicSubject || undefined;
    this.silent = options.silent || detError?.silent || false;
    this.type = options.type || detError?.type || ErrorType.Unknown;
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

const handleError = (error: unknown, options?: DetErrorOptions): void => {
  if (!isError(error) && !isDetError(error)) return;

  // Normalize error as DetError.
  const e = isError(error) ? new DetError(error, options) : error;

  // Ignore request cancellation errors.
  if (isAborted(e)) return;

  // Redirect to logout if Auth failure detected (auth token is no longer valid).`
  if (e.type === ErrorType.Auth) {
    history.push(paths.logout(), { loginRedirect: filterOutLoginLocation(window.location) });
  }

  // TODO add support and checking for saving and dismissing class of errors as user preference
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
