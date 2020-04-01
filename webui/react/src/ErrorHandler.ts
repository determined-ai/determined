import { notification } from 'antd';

import { isAsyncFunction } from 'utils/data';
import Logger, { LoggerInterface } from 'utils/Logger';

const logger = new Logger('EH');

export enum ErrorLevel {
  Fatal = 'fatal',
  Error = 'error',
  Warn = 'warning',
}

export enum ErrorType {
  Server = 'server', // internal apis and server errors.
  Ui = 'ui',
  Input = 'input',
  Api = 'api', // third-party api
}

export interface DaError {
  error?: Error;
  id?: string; // slug unique to each place in the codebase that we will use this.
  level?: ErrorLevel;
  message: string; // internal message.
  payload?: unknown;
  silent?: boolean;
  isUserTriggered?: boolean; // whether the error was caused by an active interaction.
  logger?: LoggerInterface;
  publicSubject?: string;
  publicMessage?: string;
  type: ErrorType;
}

const openNotification = (e: DaError): void => {
  const config = {
    description: e.publicMessage,
    message: e.publicSubject || `${e.type} ${e.level}`,
  };
  switch (e.level) {
    case ErrorLevel.Fatal:
    case ErrorLevel.Error:
      notification.error(config);
      break;
    case ErrorLevel.Warn:
      notification.warn(config);
      break;
  }
};

const defaultErrorParameters = {
  isUserTriggered: true,
  level: ErrorLevel.Error,
  silent: false,
};

const handleError = (e: DaError, doThrow: boolean): void => {
  // set the defaults.
  e = { ...defaultErrorParameters, ...e };

  // TODO add support and checking for saving and dismissing class of errors as user preference
  // using id.
  const skipNotification = e.silent || (e.level === ErrorLevel.Warn && !e.publicMessage);
  if (!skipNotification) openNotification(e);

  // TODO generate stack trace if error is missing? http://www.stacktracejs.com/

  // log the error if needed.
  if (!e.silent) {
    const msg = `${e.type}: ${e.publicMessage + ' ' || ''}${e.message}`;
    const targetLogger = e.logger || logger;
    switch (e.level) {
      case ErrorLevel.Fatal:
      case ErrorLevel.Error:
        targetLogger.error(msg);
        break;
      case ErrorLevel.Warn:
        targetLogger.warn(msg);
        break;
    }
  }

  // TODO SEP handle transient failures? eg only take action IF.. (requires keeping state)
  // TODO report to segment (with rate limiting? batching?). save the error stack and more for this.

  if (doThrow) {
    if (e.error !== undefined) {
      throw e;
    } else {
      throw new Error(e.message);
    }
  }

  // TODO SEP capture screen shot or more context?
  // https://stackblitz.com/edit/react-screen-capture?file=index.js
};

export const handleErrorForFn = async <T>(daError: DaError, func: () => T): Promise<T|void> => {
  try {
    return isAsyncFunction(func) ? await func() : func();
  } catch (e) {
    handleError({ ...daError, error: daError.error || e }, false);
  }
};

export default handleError;
