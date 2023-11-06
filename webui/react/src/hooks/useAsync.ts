import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { useCallback, useEffect, useInsertionEffect, useRef, useState } from 'react';

type LoadablePromiser<T> = (canceler: AbortController) => Promise<T | Loadable<T>>;

/**
 * A hook that manages the result of an async function. The async function is
 * called as a render effect with an AbortController which is aborted on
 * unmount.  While pending, the hook returns `NotLoaded`, and on complete, the
 * hook returns the return value wrapped in `Loaded`. If the function returns a
 * `Loadable`, the value isn't wrapped. When any value in the deps array
 * changes, the function is run again as a render effect. If the deps array
 * changes before the async function returns, useAsync will discard the results
 * of the older call.
 * @param loadableFunc (canceler: AbortController) => Promise<T | Loadable<T>>
 * @param deps readonly unknown[]
 * @returns Loadable<T>
 */
export const useAsync = <T>(
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
      setState(NotLoaded);
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
