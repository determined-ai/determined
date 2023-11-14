import { is, Map } from 'immutable';
import * as t from 'io-ts';

import { Json, JsonObject } from 'types';

import { asValueObject } from './asValueObject';

const jsonCodec: t.Type<Json> = t.recursion('Json', () =>
  t.union([t.string, t.number, t.boolean, t.null, t.array(jsonCodec), jsonObjectCodec]),
);
const jsonObjectCodec: t.Type<JsonObject> = t.recursion('JsonObject', () =>
  t.record(t.string, jsonCodec),
);

const codec = t.type({
  any: t.any,
  array: t.array(t.number),
  boolean: t.boolean,
  deepObject: jsonObjectCodec,
  dictionary: t.record(t.string, t.string),
  nested: t.readonly(
    t.type({
      boolean: t.boolean,
      literal: t.literal('literal'),
    }),
  ),
  number: t.number,
  string: t.string,
});

const testVal: t.TypeOf<typeof codec> = {
  any: '+',
  array: [1, 2, 3],
  boolean: true,
  deepObject: {
    cool: {
      beans: [{ bears: true }],
    },
    wow: 'woah',
  },
  dictionary: { foo: 'bar' },
  nested: {
    boolean: false,
    literal: 'literal',
  },
  number: 100,
  string: 'hewwo :3',
};

const valueObj = asValueObject(codec, testVal);
const cloneValueObj = asValueObject(codec, structuredClone(testVal));
const other = asValueObject(codec, { ...testVal, deepObject: { number: 21 } });
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const otherExtra = asValueObject(codec, { ...testVal, extra: 2 } as any);
const map = Map({ key: valueObj });

describe('asValueObject', () => {
  describe('ValueObject#hashcode', () => {
    it('returns a number for the value object', () => {
      expect(typeof valueObj.hashCode()).toBe('number');
    });

    it('returns different numbers for different value objects', () => {
      expect(valueObj.hashCode()).not.toBe(other.hashCode());
    });

    it('only includes values in the codec in the hash', () => {
      expect(valueObj.hashCode()).toBe(otherExtra.hashCode());
    });
  });
  describe('ValueObject#equals', () => {
    it('returns true if the objects are equal', () => {
      expect(valueObj.equals(cloneValueObj)).toBe(true);
      expect(cloneValueObj.equals(valueObj)).toBe(true);
    });

    it("returns false if the objects aren't equal", () => {
      expect(valueObj.equals(other)).toBe(false);
      expect(other.equals(valueObj)).toBe(false);
    });
  });

  describe('in immutable', () => {
    it('immutable.is returns true when setting to a clone', () => {
      expect(is(map, map.set('key', cloneValueObj))).toBe(true);
    });

    it('immutable.is returns false when setting to another object', () => {
      expect(is(map, map.set('key', other))).toBe(false);
    });

    it('immutable.is returns true when setting to an object with extra props', () => {
      expect(is(map, map.set('key', otherExtra))).toBe(true);
    });

    it('immutable.is falls back to value equality on fields where typechecking is disabled', () => {
      const sameAny = asValueObject(codec, { ...testVal, any: '+' });
      const newAny = asValueObject(codec, { ...testVal, any: {} });
      const newAny2 = asValueObject(codec, { ...testVal, any: {} });

      expect(is(map, map.set('key', sameAny))).toBe(true);
      expect(is(map.set('key', newAny), map.set('key', newAny2))).toBe(false);
    });
  });
});
