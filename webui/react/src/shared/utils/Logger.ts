import { debug } from 'debug';

const LIB_NAME = 'det';
export const NAMEPACE_SEPARATOR = '/';

/**
 * Log levels in order of serverity (low to high).
 * Modeled after Syslog RFC 5424
 * https://tools.ietf.org/html/rfc5424
 */
enum Level {
  Error = 'error',
  Warn = 'warn',
  Info = 'info',
  Debug = 'debug',
  Trace = 'trace',
}

// enum LogBackend {
//   Console,
//   Debug,
// }

const generateNamespace = (parts: string[], separator = NAMEPACE_SEPARATOR) => {
  return parts.join(separator);
};

const getLogger = (namespace: string, level: Level) => {
  const logger = debug(`${namespace}:${level}`);
  // debug doesn't seem to match the advertised type definition.
  return logger as (...msg: unknown[]) => void;
};

export interface LoggerInterface {
  debug(...msg: unknown[]): void;
  error(...msg: unknown[]): void;
  info(...msg: unknown[]): void;
  trace(...msg: unknown[]): void;
  warn(...msg: unknown[]): void;
}

type LogPredicate = (namespace: string, level: Level) => boolean;

/**
 * log filtering is controlled by localStorage.debug.
 * read more on: https://github.com/debug-js/debug#usage
 */
class Logger implements LoggerInterface {
  private namespace: string;
  private isVisible: LogPredicate;

  constructor(namespace: string) {
    this.namespace = namespace;
    this.isVisible = () => true;
    // debugger;
  }

  extend(...namespace: string[]): Logger {
    return new Logger(generateNamespace([ this.namespace, ...namespace ]));
  }

  /**
   * set the logic to determine whether to log each
   * message to console or not.
  */
  setVisibility(predicate: LogPredicate): void { // TODO remove me?
    this.isVisible = predicate;
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
    if (!this.isVisible(this.namespace, level)) return;
    // TODO: set up and persist loggers.
    getLogger(this.namespace, level)(...msg);
  }
}

const rootLogger = new Logger(LIB_NAME);

export default rootLogger;
