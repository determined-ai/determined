import { MemoryStore, Storage } from './storage';

const testKey = 'testKey';
const anotherTestKey = 'anotherTestKey';

describe('MemoryStore', () => {
  const testStorage = new Storage({ store: new MemoryStore() });

  beforeEach(() => {
    testStorage.clear();
  });

  it('should not have the key "testKey"', () => {
    expect(testStorage.get('testKey')).toBeNull();
  });

  it('should set "testKey" value', () => {
    const value = 'all set';
    expect(() => testStorage.set(testKey, value)).not.toThrow();
  });

  it('should get "testKey" value', () => {
    const value = 'all set';
    testStorage.set(testKey, value);
    expect(testStorage.get(testKey)).toBe(value);
  });

  it('should remove "testKey" value', () => {
    const value = 'all set';
    testStorage.set(testKey, value);
    testStorage.remove(testKey);
    expect(testStorage.get(testKey)).toBeNull();
  });

  it('should clear all values', () => {
    const value1 = { x: 1, y: 2, z: 3 };
    const value2 = [ 'a', 'b', 'c' ];
    testStorage.set(testKey, value1);
    testStorage.set(anotherTestKey, value2);
    testStorage.clear();
    expect(testStorage.get(testKey)).toBeNull();
    expect(testStorage.get(anotherTestKey)).toBeNull();
  });

  it('should work with arrays', () => {
    const value = [ 'test', 'a', 'b' ];
    testStorage.set(testKey, value);
    expect(testStorage.get(testKey)).toStrictEqual(value);
  });

  it('should work with nested arrays', () => {
    const value = [ 'test', 'a', [ 1, 2, 3 ] ];
    testStorage.set(testKey, value);
    expect(testStorage.get(testKey)).toStrictEqual(value);
  });

  it('should be able to fall back to a default', () => {
    const nonExistingKey = 'xyz';
    expect(testStorage.get(nonExistingKey)).toBeNull();
    expect(testStorage.getWithDefault(nonExistingKey, 'fallbackValue')).toBe('fallbackValue');
  });

  it('should disallow storing a Set type', () => {
    const value = new Set([ 1, 2, 3 ]);
    expect(() => testStorage.set(testKey, value)).toThrow();
  });

  it('should disallow storing a null or undefined value', () => {
    expect(() => testStorage.set(testKey, null)).toThrow();
    expect(() => testStorage.set(testKey, undefined)).toThrow();
  });
});
