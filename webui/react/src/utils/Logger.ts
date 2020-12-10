enum Level {
  Warn = 'warn',
  Error = 'error',
}

export interface LoggerInterface {
  error(msg: unknown): void;
  warn(msg: unknown): void;
}

class Logger implements LoggerInterface {
  private namespace: string;

  constructor(namespace: string) {
    this.namespace = namespace;
  }

  error(msg: unknown): void {
    this.logWithLevel(Level.Error, msg);
  }

  warn(msg: unknown): void {
    this.logWithLevel(Level.Warn, msg);
  }

  private logWithLevel(level: Level, msg: unknown): void {
    /* eslint-disable-next-line no-console */
    console[level](`[${this.namespace}]`, msg);
  }
}

export default Logger;
