import { Observable, observable, useObservable, WritableObservable } from 'micro-observables';
import React from 'react';

import { isEqual } from 'utils/data';

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

export { observable, Observable, useObservable, WritableObservable, useValueMemoizedObservable };
