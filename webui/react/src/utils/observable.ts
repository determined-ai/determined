import { Collection, is } from 'immutable';
import { isEqual } from 'lodash';
import {
  Observable,
  observable,
  Options,
  useObservable,
  WritableObservable,
} from 'micro-observables';
import React from 'react';

/**
 * Observable that does a deep equality check before changing values
 */
export class DeepObservable<T> extends WritableObservable<T> {
  override set(v: T | Observable<T>): void {
    const newValue = v instanceof Observable ? v.get() : v;
    if (!isEqual(newValue, this._evaluate())) super.set(v);
  }
}

/**
 * observable that checks equality via Immutable.is before changing values
 */
export class ImmutableObservable<
  T extends Collection<unknown, unknown>,
> extends WritableObservable<T> {
  override set(v: T | Observable<T>): void {
    const newValue = v instanceof Observable ? v.get() : v;
    if (!is(newValue, this._evaluate())) super.set(v);
  }
}

const deepObservable = <T>(v: T, o?: Options): DeepObservable<T> => new DeepObservable(v, o);

const immutableObservable = <T extends Collection<unknown, unknown>>(
  v: T,
  o?: Options,
): ImmutableObservable<T> => new ImmutableObservable(v, o);

// type comparator<T> = (current: T, previous: T) => boolean;

// Observable.prototype.listenWhile = <T>(fn: comparator<T>): void => {
//   const unsubscribe = {
//     value: null,
//   };
//   const unsub = this.onChange((value, oldValue) => {
//     if (!fn(value, oldValue) && unsubscribe.value) {
//       unsubscribe.value();
//     }
//   });
//   unsubscribe.value = unsub;
// };

const useValueMemoizedObservable = <T>(o: Observable<T>): T => {
  const [, forceRender] = React.useState({});
  const value = o.get();

  React.useEffect(() => {
    if (o.get() !== value) {
      forceRender({});
    }
    return o.subscribe((value, prevValue) => {
      if (!isEqual(value, prevValue)) {
        forceRender({});
      }
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [o]);

  return value;
};

export {
  observable,
  Observable,
  useObservable,
  WritableObservable,
  useValueMemoizedObservable,
  deepObservable,
  immutableObservable,
};
