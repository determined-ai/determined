import fc, { Arbitrary } from 'fast-check';
import { is, Map } from 'immutable';
import * as t from 'io-ts';
import { isEqual } from 'lodash';

import { Json, JsonObject } from 'types';

import { asValueObjectFactory } from './asValueObject';

const codec = t.type({
  any: t.any,
  array: t.array(t.number),
  boolean: t.boolean,
  deepObject: JsonObject,
  dictionary: t.record(t.string, t.string),
  nested: t.readonly(
    t.type({
      boolean: t.boolean,
      literal: t.literal('literal'),
    }),
  ),
  number: t.number,
  string: t.string,
  union: t.union([t.literal(true), t.literal(0)]),
});

const asCodec = asValueObjectFactory(codec);

const arb = fc.record({
  any: fc.jsonValue({ maxDepth: 0 }),
  array: fc.array(fc.float()),
  boolean: fc.boolean(),
  // weird type mismatch between json types, idk
  deepObject: fc.dictionary(fc.string(), fc.jsonValue() as fc.Arbitrary<Json>),
  dictionary: fc.dictionary(fc.string(), fc.string()),
  nested: fc.record({
    boolean: fc.boolean(),
    literal: fc.constant('literal' as const),
  }),
  number: fc.float(),
  string: fc.string(),
  union: fc.constantFrom(true as const, 0 as const),
});

const extraArb = fc.record({
  extra: fc.anything(),
});

const valueObjArb = arb.map(asCodec);

const cloneValueObjArb = fc.clone(arb, 2).map(([f, s]) => [asCodec(f), asCodec(s)] as const);

const differValueObjArb = fc
  .tuple(arb, arb)
  .filter((args) => !isEqual(...args))
  .map(([f, s]) => [asCodec(f), asCodec(s)] as const);

const extraValueObjArb = fc
  .tuple(arb, extraArb)
  .map(([testVal, extra]) => [asCodec(testVal), asCodec({ ...testVal, ...extra })] as const);

const withMapArb = <T, V extends readonly [T, T]>(a: Arbitrary<V>) =>
  a.map(([f, s]) => [Map({ key: f }), Map({ key: s })] as const);

describe('asValueObject', () => {
  describe('ValueObject#hashcode', () => {
    it.concurrent('returns a number for the value object', () => {
      fc.assert(
        fc.property(valueObjArb, (valueObj) => {
          expect(typeof valueObj.hashCode()).toBe('number');
        }),
      );
    });

    it.concurrent('returns different numbers for different value objects', () => {
      fc.assert(
        fc.property(differValueObjArb, ([testVal, other]) => {
          expect(testVal.hashCode()).not.toEqual(other.hashCode());
        }),
      );
    });

    it.concurrent('only includes values in the codec in the hash', () => {
      fc.assert(
        fc.property(extraValueObjArb, ([testObj, extraObj]) => {
          expect(testObj.hashCode()).toEqual(extraObj.hashCode());
        }),
      );
    });
  });

  describe('ValueObject#equals', () => {
    it.concurrent('returns true if the objects are equal', () => {
      fc.assert(
        fc.property(cloneValueObjArb, ([valueObj, otherObj]) => {
          expect(valueObj.equals(otherObj)).toBe(true);
          expect(otherObj.equals(valueObj)).toBe(true);
        }),
      );
    });

    it.concurrent("returns false if the objects aren't equal", () => {
      fc.assert(
        fc.property(differValueObjArb, ([valueObj, otherObj]) => {
          expect(valueObj.equals(otherObj)).toBe(false);
          expect(otherObj.equals(valueObj)).toBe(false);
        }),
      );
    });
  });

  describe('in immutable', () => {
    it.concurrent('immutable.is returns true when setting to a clone', () => {
      fc.assert(
        fc.property(withMapArb(cloneValueObjArb), ([valueMap, otherMap]) => {
          expect(is(valueMap, otherMap)).toBe(true);
        }),
      );
    });

    it.concurrent('immutable.is returns false when setting to another object', () => {
      fc.assert(
        fc.property(withMapArb(differValueObjArb), ([valueMap, otherMap]) => {
          expect(is(valueMap, otherMap)).toBe(false);
        }),
      );
    });

    it.concurrent('immutable.is returns true when setting to an object with extra props', () => {
      fc.assert(
        fc.property(withMapArb(extraValueObjArb), ([valueMap, otherMap]) => {
          expect(is(valueMap, otherMap)).toBe(true);
        }),
      );
    });
  });
});
