import { useCallback, useEffect, useInsertionEffect, useRef, useState } from 'react';

import { Loadable, Loaded, NotLoaded } from 'components/kit/utils/loadable';

type LoadablePromiser<T> = (canceler: AbortController) => Promise<T | Loadable<T>>;
export const useLoadable = <T>(
  loadableFunc: LoadablePromiser<T>,
  deps: readonly unknown[],
): Loadable<T> => {
  const [state, setState] = useState<Loadable<T>>(NotLoaded);

  // funky polyfill for useEffectEvent, which would allow us to mark the
  // function as non-reactive for the useEffect
  const funcRef = useRef<LoadablePromiser<T>>(loadableFunc);
  useInsertionEffect(() => {
    funcRef.current = loadableFunc;
  }, [loadableFunc]);
  const callFunc = useCallback(
    (canceler: AbortController) => {
      return funcRef.current(canceler);
    },
    [funcRef],
  );

  useEffect(() => {
    const internalCanceler = new AbortController();
    (async () => {
      const retVal = await callFunc(internalCanceler);
      if (!internalCanceler.signal.aborted) {
        setState(Loadable.isLoadable(retVal) ? retVal : Loaded(retVal));
      }
    })();
    return () => {
      internalCanceler.abort();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [callFunc, ...deps]);

  return state;
};
