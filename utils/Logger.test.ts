import { getLogger, Level } from './Logger';
/* eslint-disable no-console */

describe('Logger Utilities', () => {
  const namespace = 'unit-test';
  const msg = 'testMessage';

  it('debug should accept variable msg arg length', () => {
    // debug type defintions don't seem to match the library.
    const dLogger = getLogger(namespace, Level.Trace);
    expect(() => dLogger(msg, 'b', 'c')).not.toThrow();
    expect(() => dLogger(msg, 'b')).not.toThrow();
    expect(typeof dLogger).toBe('function');
  });
});
