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
const map = Map({ key: valueObj });

describe('asValueObject', () => {
  describe('ValueObject#hashcode', () => {
    it('returns a number for the value object', () => {
      expect(typeof valueObj.hashCode()).toBe('number');
    });

    it('returns different numbers for different value objects', () => {
      expect(valueObj.hashCode()).not.toBe(other.hashCode());
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
  });
});
