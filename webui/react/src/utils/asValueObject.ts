import { getEq as arrayEqFor, every, map, some, sort } from 'fp-ts/Array';
import { Eq as boolEq } from 'fp-ts/boolean';
import { Eq, fromEquals, struct as structEq } from 'fp-ts/Eq';
import { flow, pipe, tuple, tupled } from 'fp-ts/function';
import { contramap } from 'fp-ts/Ord';
import { and } from 'fp-ts/Predicate';
import { fromEntries, getEq as recordEqFor, toEntries } from 'fp-ts/Record';
import { Eq as stringEq, Ord as stringOrd } from 'fp-ts/string';
import { fst, mapSnd } from 'fp-ts/Tuple';
import { hash } from 'immutable';
import * as t from 'io-ts';

// fp-ts's equality checking for nubmers fails if both values are NaN. use the
// SameValueZero algorithm to determine equality instead
const numberEq = fromEquals((x: number, y: number) => [x].includes(y));

export type ValueObjectOf<T> = T extends { equals?: unknown; hashCode?: unknown }
  ? never
  : T & {
      equals: (other: unknown) => boolean;
      hashCode: () => number;
    };

export const getProps = <T extends t.HasProps>(codec: T): t.Props => {
  switch (codec._tag) {
    case 'RefinementType':
    case 'ReadonlyType':
      return getProps(codec.type);
    case 'StrictType':
    case 'PartialType':
    case 'InterfaceType':
      return codec.props;
    case 'IntersectionType':
      return codec.types.reduce((acc, type) => ({ ...acc, ...getProps(type) }), {});
  }
};

const isHasProps = (codec: t.Mixed): codec is t.HasProps => {
  return (
    codec instanceof t.StrictType ||
    codec instanceof t.PartialType ||
    codec instanceof t.InterfaceType ||
    ((codec instanceof t.RefinementType || codec instanceof t.ReadonlyType) &&
      isHasProps(codec.type)) ||
    (codec instanceof t.IntersectionType && codec.types.every(isHasProps))
  );
};

const wrapCodecEq = <T extends t.Mixed>(codec: T, eq: Eq<t.TypeOf<T>>) =>
  fromEquals(flow(tuple<[unknown, unknown]>, pipe(every(codec.is), pipe(eq.equals, tupled, and))));

const mixedToEq = <T extends t.Mixed>(codec: T): Eq<t.TypeOf<T>> => {
  if (isHasProps(codec)) {
    // get the equality of each prop in the codec
    const props = getProps(codec);
    return pipe(
      props,
      toEntries,
      // can't use as const here because fromEntries wants a writable
      map(([key, c]): [string, Eq<t.TypeOf<t.Mixed>>] => [key, wrapCodecEq(c, mixedToEq(c))]),
      fromEntries,
      structEq,
    );
  }
  if (codec instanceof t.RecursiveType) {
    // use the thunk to lazily get the equality as we recurse
    return wrapCodecEq(
      codec,
      fromEquals((x: unknown, y: unknown) => mixedToEq(codec.runDefinition()).equals(x, y)),
    );
  }
  if (codec instanceof t.ExactType || codec instanceof t.ReadonlyType) {
    // get the equality of the wrapped type
    return wrapCodecEq(codec, mixedToEq(codec.type));
  }
  if (codec instanceof t.UnionType) {
    // check to see if there's any codec where both values pass the typecheck and are equal to each other
    return wrapCodecEq(
      codec,
      fromEquals(
        flow(
          tuple<[unknown, unknown]>,
          pipe(
            codec.types,
            map(mixedToEq),
            (eqs) => (args) =>
              pipe(
                eqs,
                some((eq) => eq.equals(...args)),
              ),
          ),
        ),
      ),
    );
  }
  if (codec instanceof t.DictionaryType) {
    // not sure how this behaves with number/symbol keys
    return wrapCodecEq(codec, recordEqFor(mixedToEq(codec.codomain)));
  }
  if (codec instanceof t.ArrayType || codec instanceof t.ReadonlyArrayType) {
    return wrapCodecEq(codec, arrayEqFor(mixedToEq(codec.type)));
  }
  if (codec instanceof t.LiteralType) {
    switch (typeof codec.value) {
      case 'string': {
        return wrapCodecEq(codec, stringEq);
      }
      case 'number': {
        return wrapCodecEq(codec, numberEq);
      }
      case 'boolean': {
        return wrapCodecEq(codec, boolEq);
      }
      default: {
        // this shouldn't be reachable, but is here because the type guard
        // resolves codec to LiteralType<any> and that means value is any
        // instead of string | number | boolean
        return wrapCodecEq(
          codec,
          fromEquals((x, y) => x === y),
        );
      }
    }
  }
  if (codec instanceof t.StringType) {
    return wrapCodecEq(codec, stringEq);
  }
  if (codec instanceof t.NumberType) {
    return wrapCodecEq(codec, numberEq);
  }
  if (codec instanceof t.BooleanType) {
    return wrapCodecEq(codec, boolEq);
  }
  // fall back to triple equals
  return wrapCodecEq(
    codec,
    fromEquals((x, y) => x === y),
  );
};

// clone an object, only grabbing values defined in the codec. similar to
// exact(codec).encode(value), but recursive
const recursiveStripKeys = <T extends t.Mixed>(codec: T, value: t.TypeOf<T>): t.TypeOf<T> => {
  if (isHasProps(codec)) {
    const props = getProps(codec);
    return pipe(
      props,
      toEntries,
      map(([key, c]): [string, t.TypeOf<t.Mixed>] => [key, recursiveStripKeys(c, value[key])]),
      fromEntries,
    );
  }
  if (
    codec instanceof t.ExactType ||
    codec instanceof t.ReadonlyType ||
    codec instanceof t.RecursiveType
  ) {
    return recursiveStripKeys(codec.type, value);
  }
  if (codec instanceof t.DictionaryType) {
    // return object with keys in stable order
    return pipe(
      value,
      toEntries,
      map(mapSnd((v) => recursiveStripKeys(codec.codomain, v))),
      sort(pipe(stringOrd, contramap(fst<string, unknown>))),
      fromEntries,
    );
  }
  if (codec instanceof t.ArrayType || codec instanceof t.ReadonlyArrayType) {
    return pipe(
      value,
      map((v) => recursiveStripKeys(codec.type, v)),
    );
  }
  return value;
};

/**
 * ValueObject is an interface that comes from immutable. ValueObjects
 * implmement a hash code for quick checks and a equals function for checking
 * against other values. This function converts values using a codec to
 * ValueObjects.
 *
 * NOTE: ValueObject methods are only used when checking equality via
 * `Immutable.is` -- setting a key pointing to a ValueObject to an equal
 * ValueObject will still return a new object reference.
 */
export const asValueObject = <T extends t.HasProps>(
  codec: T,
  value: t.TypeOf<T>,
): ValueObjectOf<t.TypeOf<T>> => {
  const eq = mixedToEq(codec);
  return {
    ...value,
    equals(other: unknown) {
      return eq.equals(this, other);
    },
    hashCode() {
      return hash(JSON.stringify(recursiveStripKeys(codec, this)));
    },
  };
};

/**
 * convenience function for asValueObject when reusing the same codec
 */
export const asValueObjectFactory =
  <T extends t.HasProps>(codec: T) =>
  (value: t.TypeOf<T>): ValueObjectOf<t.TypeOf<T>> =>
    asValueObject(codec, value);

export default asValueObject;
