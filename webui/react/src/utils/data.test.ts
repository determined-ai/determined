import {
  clone,
  isAsyncFunction,
  isFunction,
  isMap,
  isNumber,
  isObject,
  isPrimitive,
  isSet,
  isSyncFunction,
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

describe('data utility', () => {
  const object = { a: true, b: null, c: { x: { y: -1.2e10 }, z: undefined } };

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

  testGroups.forEach(group => {
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
});
