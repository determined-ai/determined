import Logger from './Logger';
/* eslint-disable no-console */

describe('Logger Utilities', () => {
  const namespace = 'unit-test';
  const msg = 'testMessage';
  const logger = new Logger(namespace);
  const original = {
    error: console.error,
    warn: console.warn,
  };

  beforeEach(() => {
    console.error = jest.fn();
    console.warn = jest.fn();
  });

  afterEach(() => {
    console.error = original.error;
    console.warn = original.warn;
  });

  it('should log to correct log level', () => {
    logger.error(msg);
    expect(console.error).toHaveBeenCalledTimes(1);
    expect(console.warn).toHaveBeenCalledTimes(0);

    logger.warn(msg);
    expect(console.error).toHaveBeenCalledTimes(1);
    expect(console.warn).toHaveBeenCalledTimes(1);
  });

  it('should format the message as "[namescpae] msg"', () => {
    logger.error(msg);
    expect(console.error).toHaveBeenCalledWith(`[${namespace}]`, msg);

    logger.warn(msg);
    expect(console.warn).toHaveBeenCalledWith(`[${namespace}]`, msg);
  });
});
