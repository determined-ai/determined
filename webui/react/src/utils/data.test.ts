import testConfig from 'fixtures/responses/trial-details/old-trial-config-noop-adaptive.json';
import { RawJson } from 'types';

import {
  clone,
  deletePathList,
  getPath,
  getPathList,
  getPathOrElse,
  isAsyncFunction,
  isFunction,
  isMap,
  isNumber,
  isObject,
  isPrimitive,
  isSet,
  isSyncFunction,
  numericSorter,
  setPathList,
} from './data';

enum Type {
  AsyncFn = 'async-function',
  Fn = 'function',
  Map = 'map',
  Number = 'number',
  Object = 'object',
  Primitive = 'primitive',
  Set = 'set',
  SyncFn = 'sync-function',
}

const testGroups = [
  { fn: isAsyncFunction, type: Type.AsyncFn },
  { fn: isFunction, type: Type.Fn },
  { fn: isMap, type: Type.Map },
  { fn: isNumber, type: Type.Number },
  { fn: isObject, type: Type.Object },
  { fn: isPrimitive, type: Type.Primitive },
  { fn: isSet, type: Type.Set },
  { fn: isSyncFunction, type: Type.SyncFn },
];

/* eslint-disable-next-line @typescript-eslint/no-empty-function */
const voidFn = (): void => {};

const syncFn = (): Promise<boolean> => {
  return new Promise((resolve) => {
    setTimeout(() => resolve(true), 10);
  });
};

const asyncFn = async (): Promise<boolean> => {
  try {
    const response = await syncFn();
    return response;
  } catch (e) {
    voidFn();
    throw e;
  }
};

const tests = [
  { type: [ Type.AsyncFn, Type.Fn ], value: asyncFn },
  { type: [ Type.SyncFn, Type.Fn ], value: syncFn },
  { type: [ Type.SyncFn, Type.Fn ], value: voidFn },
  { type: [ Type.Map, Type.Object ], value: new Map() },
  { type: [ Type.Map, Type.Object ], value: new Map([ [ 'a', 'value1' ], [ 'b', 'value2' ] ]) },
  { type: [ Type.Map, Type.Object ], value: new Map([ [ 'x', -1 ], [ 'y', 1.5 ] ]) },
  { type: Type.Primitive, value: 'Jalapeño' },
  { type: [ Type.Number, Type.Primitive ], value: -3.14159 },
  { type: [ Type.Number, Type.Primitive ], value: 1.23e-8 },
  { type: [ Type.Number, Type.Primitive ], value: 0 },
  { type: Type.Primitive, value: null },
  { type: Type.Primitive, value: undefined },
  { type: Type.Object, value: {} },
  { type: Type.Object, value: { 0: 1.5, a: undefined, [Symbol('b')]: null } },
  { type: [ Type.Set, Type.Object ], value: new Set() },
  { type: [ Type.Set, Type.Object ], value: new Set([ 'abc', 'def', 'ghi' ]) },
  { type: [ Type.Set, Type.Object ], value: new Set([ -1.5, Number.MAX_VALUE, null, undefined ]) },
];
const object = { a: true, b: null, c: { x: { y: -1.2e10 }, z: undefined } };

describe('data utility', () => {
  describe('clone', () => {
    it('should preserve primitives', () => {
      expect(clone(-1.23e-8)).toBe(-1.23e-8);
      expect(clone(0)).toBe(0);
      expect(clone('Jalapeño')).toBe('Jalapeño');
      expect(clone(false)).toBe(false);
      expect(clone(false)).toBe(false);
      expect(clone(null)).toBeNull();
      expect(clone(undefined)).toBeUndefined();
    });

    it('should clone maps', () => {
      const map = new Map([ [ 'x', -1 ], [ 'y', 1.5 ] ]);
      expect(clone(map)).not.toBe(map);
      expect(clone(map)).toMatchObject(map);
    });

    it('should clone sets', () => {
      const set = new Set([ -1.5, Number.MAX_VALUE, null, undefined ]);
      expect(clone(set)).not.toBe(set);
      expect(clone(set)).toMatchObject(set);
    });

    it('should clone shallow objects', () => {
      const shallowClone = clone(object, false);
      expect(shallowClone).not.toBe(object);
      expect(shallowClone.c).toBe(object.c);
      expect(shallowClone.c.x).toBe(object.c.x);
    });

    it('should clone deep objects', () => {
      const deepClone = clone(object);
      expect(deepClone).not.toBe(object);
      expect(deepClone.c).not.toBe(object.c);
      expect(deepClone.c.x).not.toBe(object.c.x);
    });
  });

  describe('getPath', () => {
    it('should get object value based on paths', () => {
      expect(getPath<boolean>(object, 'a')).toBe(true);
      expect(getPath<string>(object, 'x.x')).toBeUndefined();
      expect(getPath<number>(object, 'c.x.y')).toBe(-1.2e10);
    });

    it('should support empty path', () => {
      expect(getPath<RawJson>(object, '')).toBe(object);
    });

  });

  describe('getPathOrElse', () => {
    it('should get-or-else objects', () => {
      expect(getPathOrElse<boolean>(object, 'a', false)).toBe(true);
      expect(getPathOrElse<string>(object, 'b', 'junk')).toBeNull();
      expect(getPathOrElse<number>(object, 'c.x.y', 0)).toBe(-1.2e10);
    });

    it('should get-or-else fallbacks', () => {
      const fallback = 'fallback';
      expect(getPathOrElse<string>(object, 'a.b.c', fallback)).toBe(fallback);
      expect(getPathOrElse<string>(object, 'c.x.w', fallback)).toBe(fallback);
      expect(getPathOrElse<string | undefined>(object, 'c.x.z', undefined)).toBeUndefined();
    });
  });

  describe('chained object manipulators', () => {
    let config = clone(testConfig);

    beforeAll(() => {
      config = clone(testConfig);
    });

    describe('getPathList', () => {
      it('should return undefined for bad paths', () => {
        const actual = getPathList(config, [ 'x', 'y', 'z' ]);
        expect(actual).toBeUndefined();
      });

      it('should return undefined for partial matching bad paths', () => {
        const path = [ 'searcher', 'step_budget' ];
        expect(getPathList(config, path)).not.toBeUndefined();
        const actual = getPathList(config, [ ...path, 'xyz' ]);
        expect(actual).toBeUndefined();
      });

      it('should return null', () => {
        const actual = getPathList(config, [ 'min_checkpoint_period' ]);
        expect(actual).toBeNull();
      });

      it('should return objects', () => {
        const actual = getPathList(config, [ 'searcher' ]);
        expect(actual).toHaveProperty('mode');
        expect(typeof actual).toEqual('object');
      });

      it('should return a reference', () => {
        const searcher = getPathList<RawJson>(config, [ 'searcher' ]);
        const TEST_VALUE = 'TEST';
        expect(searcher).toHaveProperty('mode');
        config.searcher.mode = TEST_VALUE;
        expect(config.searcher.mode).toEqual(TEST_VALUE);
      });
    });

    describe('deleteSubObject', () => {
      it('should remove from input', () => {
        expect(config.min_validation_period).not.toBeUndefined();
        deletePathList(config, [ 'min_validation_period' ]);
        expect(config.min_validation_period).toBeUndefined();
      });
    });

    describe('setSubObject', () => {
      it('should set on input', () => {
        const value = { abc: 3 };
        setPathList(config, [ 'min_validation_period' ], value);
        expect(config.min_validation_period).toStrictEqual(value);
        expect(config.min_validation_period === value).toBeTruthy();
      });
    });

  });

  testGroups.forEach(group => {
    /* eslint-disable-next-line jest/valid-title */
    describe(group.fn.name, () => {
      it('should test each type correctly', () => {
        tests.forEach(test => {
          const result = Array.isArray(test.type)
            ? test.type.includes(group.type)
            : test.type === group.type;
          expect(group.fn(test.value)).toBe(result);
        });
      });
    });
  });

  describe('sorters', () => {
    it('numericSorter', () => {
      const tests = [
        { input: [ 0, 1 ], output: -1 },
        { input: [ 1, 0 ], output: 1 },
        { input: [ 0, -0.123 ], output: 1 },
        { input: [ -0.123, 0 ], output: -1 },
        { input: [ 100, 10 ], output: 1 },
        { input: [ -100, -10 ], output: -1 },
        { input: [ 1, 10 ], output: -1 },
        { input: [ -1, -10 ], output: 1 },
        { input: [ 0.01, 0.1 ], output: -1 },
        { input: [ -0.01, -0.1 ], output: 1 },
        { input: [ 1.23e2, -123 ], output: 1 },
        { input: [ -0.999, 9e-3 ], output: -1 },
        { input: [ 0.123, undefined ], output: 1 },
        { input: [ 0, undefined ], output: 1 },
        { input: [ undefined, -0.123 ], output: -1 },
        { input: [ undefined, 0 ], output: -1 },
        { input: [ undefined, undefined ], output: 0 },
        { input: [ 0, 0 ], output: 0 },
        { input: [ 1e7, 1e7 ], output: 0 },
        { input: [ 1e-5, 1e-5 ], output: 0 },
        { input: [ 10, 10 ], output: 0 },
        { input: [ -0.123, -0.123 ], output: 0 },
      ];
      tests.forEach(test => {
        const result = numericSorter(test.input[0], test.input[1], false);
        const reverseResult = numericSorter(test.input[0], test.input[1], true);
        expect(result).toStrictEqual(test.output);
        expect(reverseResult).toStrictEqual(test.output === 0 ? 0 : test.output * -1);
      });
    });
  });
});
