enum Level {
  Warn  = 'warn',
  Error = 'error',
}

class Logger {
  private namespace: string;

  constructor(namespace: string) {
    this.namespace = namespace;
  }

  private logWithLevel(level: Level, msg: unknown): void {
    /* eslint-disable-next-line no-console */
    console[level](`[${this.namespace}]`, msg);
  }

  error(msg: unknown): void {
    this.logWithLevel(Level.Error, msg);
  }

  warn(msg: unknown): void {
    this.logWithLevel(Level.Warn, msg);
  }
}

export default Logger;
