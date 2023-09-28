import { debug } from 'debug';

import { ValueOf } from 'components/kit/internal/types';

const LIB_NAME = 'det';
export const NAMEPACE_SEPARATOR = '/';

/**
 * Log levels in order of serverity (low to high).
 * Modeled after Syslog RFC 5424
 * https://tools.ietf.org/html/rfc5424
 */
export const Level = {
  Debug: 'debug',
  Error: 'error',
  Info: 'info',
  Trace: 'trace',
  Warn: 'warn',
} as const;

export type Level = ValueOf<typeof Level>;

// enum LogBackend {
//   Console,
//   Debug,
// }

const generateNamespace = (parts: string[], separator = NAMEPACE_SEPARATOR) => {
  return parts.join(separator);
};

const loggers: Record<string, debug.Debugger> = {};

/** returns the underlying Debug logger. */
export const getLogger = (namespace: string, level: Level): ((...msg: unknown[]) => void) => {
  const key = `${namespace}:${level}`;
  if (!loggers[key]) {
    loggers[key] = debug(key);
  }
  return loggers[key];
};

export interface LoggerInterface {
  debug(...msg: unknown[]): void;
  error(...msg: unknown[]): void;
  info(...msg: unknown[]): void;
  trace(...msg: unknown[]): void;
  warn(...msg: unknown[]): void;
}

/**
 * log filtering is controlled by localStorage.debug.
 * read more on: https://github.com/debug-js/debug#usage
 */
class Logger implements LoggerInterface {
  private namespace: string;

  constructor(namespace: string) {
    this.namespace = namespace;
  }

  extend(...namespace: string[]): Logger {
    return new Logger(generateNamespace([this.namespace, ...namespace]));
  }

  debug(...msg: unknown[]): void {
    this.logWithLevel(Level.Debug, ...msg);
  }

  trace(...msg: unknown[]): void {
    this.logWithLevel(Level.Trace, ...msg);
  }

  info(...msg: unknown[]): void {
    this.logWithLevel(Level.Info, ...msg);
  }

  error(...msg: unknown[]): void {
    this.logWithLevel(Level.Error, ...msg);
  }

  warn(...msg: unknown[]): void {
    this.logWithLevel(Level.Warn, ...msg);
  }

  private logWithLevel(level: Level, ...msg: unknown[]): void {
    // TODO: set up and persist loggers.
    getLogger(this.namespace, level)(...msg);
  }
}

const rootLogger = new Logger(LIB_NAME);

export default rootLogger;
