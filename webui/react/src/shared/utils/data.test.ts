import testConfig from 'shared/tests/fixtures/old-trial-config-response.json';

import { RawJson, UnknownRecord } from '../types';

import * as utils from './data';

enum Type {
  AsyncFn = 'async-function',
  BigInt = 'bigint',
  Boolean = 'boolean',
  Date = 'date',
  Fn = 'function',
  Map = 'map',
  NullOrUndefined = 'null-or-undefined',
  Number = 'number',
  Object = 'object',
  Primitive = 'primitive',
  Promise = 'promise',
  Set = 'set',
  String = 'string',
  SyncFn = 'sync-function',
}

const testGroups = [
  { fn: utils.isAsyncFunction, type: Type.AsyncFn },
  { fn: utils.isBigInt, type: Type.BigInt },
  { fn: utils.isBoolean, type: Type.Boolean },
  { fn: utils.isDate, type: Type.Date },
  { fn: utils.isFunction, type: Type.Fn },
  { fn: utils.isMap, type: Type.Map },
  { fn: utils.isNullOrUndefined, type: Type.NullOrUndefined },
  { fn: utils.isNumber, type: Type.Number },
  { fn: utils.isObject, type: Type.Object },
  { fn: utils.isPrimitive, type: Type.Primitive },
  { fn: utils.isPromise, type: Type.Promise },
  { fn: utils.isSet, type: Type.Set },
  { fn: utils.isString, type: Type.String },
  { fn: utils.isSyncFunction, type: Type.SyncFn },
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

const object = { a: true, b: null, c: { x: { y: -1.2e10 }, z: undefined } };

describe('Data Utilities', () => {
  describe('type checking utilities', () => {
    const tests = [
      // Functions
      { type: [ Type.AsyncFn, Type.Fn ], value: asyncFn },
      { type: [ Type.SyncFn, Type.Fn ], value: syncFn },
      { type: [ Type.SyncFn, Type.Fn ], value: voidFn },

      // Maps and Sets
      { type: [ Type.Map, Type.Object ], value: new Map() },
      { type: [ Type.Map, Type.Object ], value: new Map([ [ 'a', 'value1' ], [ 'b', 'value2' ] ]) },
      { type: [ Type.Map, Type.Object ], value: new Map([ [ 'x', -1 ], [ 'y', 1.5 ] ]) },
      { type: [ Type.Set ], value: new Set() },
      { type: [ Type.Set ], value: new Set([ 'abc', 'def', 'ghi' ]) },
      {
        type: [ Type.Set ],
        value: new Set([ -1.5, Number.MAX_VALUE, null, undefined ]),
      },

      // Primitives
      { type: [ Type.NullOrUndefined, Type.Primitive ], value: null },
      { type: [ Type.NullOrUndefined, Type.Primitive ], value: undefined },
      { type: [ Type.BigInt, Type.Primitive ], value: 9007199254740993n },
      { type: [ Type.Number, Type.Primitive ], value: -3.14159 },
      { type: [ Type.Number, Type.Primitive ], value: 1.23e-8 },
      { type: [ Type.Number, Type.Primitive ], value: 0 },
      { type: [ Type.Primitive, Type.String ], value: 'Jalapeño' },

      // Objects
      { type: [ Type.Date, Type.Object ], value: new Date() },
      { type: Type.Object, value: {} },
      { type: Type.Object, value: { 0: 1.5, a: undefined, [Symbol('b')]: null } },
      { type: [ Type.Primitive, Type.String ], value: 'hello world' },
      { type: [ Type.Object, Type.Promise ], value: new Promise(resolve => resolve(undefined)) },
    ];
    testGroups.forEach(group => {
      describe(group.fn.name, () => {
        tests.forEach(test => {
          it(`should test value "${test.value}" correctly as ${JSON.stringify(test.type)}`, () => {
            const result = Array.isArray(test.type)
              ? test.type.includes(group.type)
              : test.type === group.type;
            expect(group.fn(test.value)).toBe(result);
          });
        });
      });
    });
  });

  describe('isEqual', () => {
    const tests = [
      { input: [ object, object ], output: true },
      {
        input: [
          { abc: 123 },
          { abc: 123, ghi: undefined },
        ],
        output: true,
      },
      {
        input: [
          { abc: 123 },
          { abc: 123, ghi: 456 },
        ],
        output: false,
      },
      {
        input: [
          { abc: 123 },
          { abc: 123, ghi: null },
        ],
        output: false,
      },
      { input: [ 'abc', 'abc' ], output: true },
      { input: [ 'abc', 'xyz' ], output: false },
      { input: [ 'abc', 0.5 ], output: false },
      { input: [ 1e-12, 1e-12 ], output: true },
      { input: [ 1e-12, 0.5 ], output: false },
      { input: [ 1e-12, 'abc' ], output: false },
      { input: [ Symbol('coin'), Symbol('coin') ], output: true },
      { input: [ Symbol('bit'), Symbol('coin') ], output: false },
      { input: [ Symbol('bit'), 123 ], output: false },
      {
        input: [
          new Set([ 'abc', 123, Symbol('xyz') ]),
          new Set([ 'abc', 123, Symbol('xyz') ]),
        ],
        output: true,
      },
      { input: [ new Set([ 'abc', 123 ]), new Set([ 'abc' ]) ], output: false },
      { input: [ new Set([ 'abc', 123 ]), new Map([ [ 'abc', 123 ] ]) ], output: false },
      {
        input: [
          new Map([ [ 'abc', 123 ], [ 'def', 456 ] ]),
          new Map([ [ 'abc', 123 ], [ 'def', 456 ] ]),
        ],
        output: true,
      },
      {
        input: [
          new Map([ [ 'abc', 123 ] ]),
          new Map([ [ 'abc', 123 ], [ 'def', 456 ] ]),
        ],
        output: false,
      },
    ];
    tests.forEach(test => {
      const [ a, b ] = test.input;
      const aLabel = utils.isPrimitive(a) ? String(a) : JSON.stringify(a);
      const bLabel = utils.isPrimitive(b) ? String(b) : JSON.stringify(b);
      it(
        `${JSON.stringify(aLabel)} should
          ${test.output ? 'not ' : ''}equal
          ${JSON.stringify(bLabel)}`,
        () => expect(utils.isEqual(a, b)).toBe(test.output),
      );
    });
  });

  describe('clone', () => {
    it('should preserve primitives', () => {
      expect(utils.clone(-1.23e-8)).toBe(-1.23e-8);
      expect(utils.clone(0)).toBe(0);
      expect(utils.clone('Jalapeño')).toBe('Jalapeño');
      expect(utils.clone(false)).toBe(false);
      expect(utils.clone(false)).toBe(false);
      expect(utils.clone(null)).toBeNull();
      expect(utils.clone(undefined)).toBeUndefined();
    });

    it('should clone maps', () => {
      const map = new Map([ [ 'x', -1 ], [ 'y', 1.5 ] ]);
      expect(utils.clone(map)).not.toBe(map);
      expect(utils.clone(map)).toMatchObject(map);
    });

    it('should clone sets', () => {
      const set = new Set([ -1.5, Number.MAX_VALUE, null, undefined ]);
      expect(utils.clone(set)).not.toBe(set);
      expect(utils.clone(set)).toMatchObject(set);
    });

    it('should clone shallow objects', () => {
      const shallowClone = utils.clone(object, false);
      expect(shallowClone).not.toBe(object);
      expect(shallowClone.c).toBe(object.c);
      expect(shallowClone.c.x).toBe(object.c.x);
    });

    it('should clone deep objects', () => {
      const deepClone = utils.clone(object);
      expect(deepClone).not.toBe(object);
      expect(deepClone.c).not.toBe(object.c);
      expect(deepClone.c.x).not.toBe(object.c.x);
    });
  });

  describe('flattenObject and unflattenObject', () => {
    const continueFn = (value: unknown) => !(value as { type: string }).type;
    const tests = [
      {
        input: {
          a: {
            x: true,
            y: -5.280,
            z: { hello: 'world' },
          },
          b: [ 0, 1, 2 ],
        },
        output: {
          'a.x': true,
          'a.y': -5.280,
          'a.z.hello': 'world',
          'b': [ 0, 1, 2 ],
        },
      },
      {
        input: {
          a: {
            x: true,
            y: -5.280,
            z: { hello: 'world' },
          },
          b: [ 0, 1, 2 ],
        },
        options: { delimiter: '->]X[<-' },
        output: {
          'a->]X[<-x': true,
          'a->]X[<-y': -5.280,
          'a->]X[<-z->]X[<-hello': 'world',
          'b': [ 0, 1, 2 ],
        },
      },
      {
        input: {
          arch: {
            n_filters1: { maxval: 64, minval: 8, type: 'int' },
            n_filters2: { maxval: 72, minval: 8, type: 'int' },
          },
          dropout1: { maxval: 0.8, minval: 0.2, type: 'double' },
          dropout2: { maxval: 0.8, minval: 0.2, type: 'double' },
          global_batch_size: { type: 'const', val: 64 },
          learning_rate: { maxval: 1, minval: 0.0001, type: 'double' },
        },
        options: { continueFn },
        output: {
          'arch.n_filters1': { maxval: 64, minval: 8, type: 'int' },
          'arch.n_filters2': { maxval: 72, minval: 8, type: 'int' },
          'dropout1': { maxval: 0.8, minval: 0.2, type: 'double' },
          'dropout2': { maxval: 0.8, minval: 0.2, type: 'double' },
          'global_batch_size': { type: 'const', val: 64 },
          'learning_rate': { maxval: 1, minval: 0.0001, type: 'double' },
        },
      },
    ];

    it('should flatten object', () => {
      tests.forEach(test => {
        expect(utils.flattenObject(test.input, test.options)).toStrictEqual(test.output);
      });
    });

    it('should unflatten object', () => {
      tests.forEach(test => {
        expect(utils.unflattenObject(test.output as UnknownRecord, test.options?.delimiter))
          .toStrictEqual(test.input);
      });
    });
  });

  describe('object path utilities', () => {
    describe('getPath', () => {
      it('should get object value based on paths', () => {
        expect(utils.getPath<boolean>(object, 'a')).toBe(true);
        expect(utils.getPath<string>(object, 'x.x')).toBeUndefined();
        expect(utils.getPath<number>(object, 'c.x.y')).toBe(-1.2e10);
      });

      it('should support empty path', () => {
        expect(utils.getPath<RawJson>(object, '')).toBe(object);
      });
    });

    describe('getPathOrElse', () => {
      it('should get-or-else objects', () => {
        expect(utils.getPathOrElse<boolean>(object, 'a', false)).toBe(true);
        expect(utils.getPathOrElse<string>(object, 'b', 'junk')).toBeNull();
        expect(utils.getPathOrElse<number>(object, 'c.x.y', 0)).toBe(-1.2e10);
      });

      it('should get-or-else fallbacks', () => {
        const fallback = 'fallback';
        expect(utils.getPathOrElse<string>(object, 'a.b.c', fallback)).toBe(fallback);
        expect(utils.getPathOrElse<string>(object, 'c.x.w', fallback)).toBe(fallback);
        expect(utils.getPathOrElse<string | undefined>(object, 'c.x.z', undefined)).toBeUndefined();
      });
    });

    describe('chained object manipulators', () => {
      let config = utils.clone(testConfig);

      beforeAll(() => {
        config = utils.clone(testConfig);
      });

      describe('getPathList', () => {
        it('should return undefined for bad paths', () => {
          const actual = utils.getPathList(config, [ 'x', 'y', 'z' ]);
          expect(actual).toBeUndefined();
        });

        it('should return undefined for partial matching bad paths', () => {
          const path = [ 'searcher', 'step_budget' ];
          expect(utils.getPathList(config, path)).not.toBeUndefined();
          const actual = utils.getPathList(config, [ ...path, 'xyz' ]);
          expect(actual).toBeUndefined();
        });

        it('should return null', () => {
          const actual = utils.getPathList(config, [ 'min_checkpoint_period' ]);
          expect(actual).toBeNull();
        });

        it('should return objects', () => {
          const actual = utils.getPathList(config, [ 'searcher' ]);
          expect(actual).toHaveProperty('mode');
          expect(typeof actual).toEqual('object');
        });

        it('should return a reference', () => {
          const searcher = utils.getPathList<RawJson>(config, [ 'searcher' ]);
          const TEST_VALUE = 'TEST';
          expect(searcher).toHaveProperty('mode');
          config.searcher.mode = TEST_VALUE;
          expect(config.searcher.mode).toEqual(TEST_VALUE);
        });
      });

      describe('deletePathList', () => {
        it('should remove from input', () => {
          expect(config.min_validation_period).not.toBeUndefined();
          utils.deletePathList(config, [ 'min_validation_period' ]);
          expect(config.min_validation_period).toBeUndefined();
        });
      });

      describe('setPathList', () => {
        it('should set on input', () => {
          const value = { abc: 3 };
          utils.setPathList(config, [ 'min_validation_period' ], value);
          expect(config.min_validation_period).toStrictEqual(value);
          expect(config.min_validation_period === value).toBeTruthy();
        });
      });
    });
  });

  describe('enum utilities', () => {
    enum CarType {
      Convertible = 'Convertible',
      Coupe = 'Coupe',
      Hatchback = 'Hatchback',
      Minivan = 'Minivan',
      PickupTruck = 'Pickup Truck',
      Sedan = 'Sedan',
      SportsCar = 'Sports Car',
      StationWagon = 'Station Wagon',
      SUV = 'SUV',
    }
    const INVALID_CAR_TYPE = 'Not a CarType';

    describe('validateEnum', () => {
      const tests = [
        { input: CarType.Convertible, output: CarType.Convertible },
        { input: CarType.Minivan, output: CarType.Minivan },
        { input: CarType.SUV, output: CarType.SUV },
        { input: INVALID_CAR_TYPE, output: undefined },
      ];
      tests.forEach(test => {
        it(`"${test.input}" should ${test.output ? '' : 'not be '}valid`, () => {
          expect(utils.validateEnum(CarType, test.input)).toBe(test.output);
        });
      });
    });

    describe('validateEnumList', () => {
      const tests = [
        {
          input: undefined,
          output: undefined,
          testName: '`undefined` enum should remain `undefined`',
        },
        {
          input: [ CarType.Coupe, CarType.PickupTruck ],
          output: [ CarType.Coupe, CarType.PickupTruck ],
          testName: 'should leave valid enum values untouched',
        },
        {
          input: [ CarType.Hatchback, INVALID_CAR_TYPE, CarType.SportsCar ],
          output: [ CarType.Hatchback, CarType.SportsCar ],
          testName: 'should filter out invalid enum values',
        },
      ];
      tests.forEach(test => {
        it(test.testName, () => {
          expect(utils.validateEnumList(CarType, test.input)).toEqual(test.output);
        });
      });
    });
  });
});
