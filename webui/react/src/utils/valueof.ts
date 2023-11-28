import { failure, identity, success, Type } from 'io-ts';

export type ValueOf<T> = T[keyof T];
export class ValueOfType<D extends { [key: string]: unknown }> extends Type<ValueOf<D>> {
  readonly _tag = 'ValueofType' as const;
  constructor(
    name: string,
    is: ValueOfType<D>['is'],
    validate: ValueOfType<D>['validate'],
    encode: ValueOfType<D>['encode'],
    readonly values: D,
  ) {
    super(name, is, validate, encode);
  }
}
// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface ValueOfC<D extends { [key: string]: unknown }> extends ValueOfType<D> {}

/**
 * Generate a codec describing a union type comprising an object's values. Like the
 * `keyof` function from io-ts, but for values instead of keys.
 */
export const valueof = <D extends { [key: string]: unknown }>(
  values: D,
  name: string = Object.values(values)
    .map((k) => JSON.stringify(k))
    .join(' | '),
): ValueOfC<D> => {
  const valueSet = new Set(Object.values(values));
  const is = (u: unknown): u is ValueOf<D> => valueSet.has(u);
  return new ValueOfType(
    name,
    is,
    (u, c) => (is(u) ? success(u) : failure(u, c)),
    identity,
    values,
  );
};

export default valueof;
