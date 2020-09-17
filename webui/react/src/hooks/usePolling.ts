import { useCallback, useEffect, useMemo, useRef } from 'react';

import { isAsyncFunction } from 'utils/data';

const DEFAULT_DELAY = 5000;

type PollingFn = (() => Promise<void>) | (() => void);

interface PollingOptions {
  delay?: number;
  triggers?: unknown[];
}

const usePolling = (pollingFn: PollingFn, { delay }: PollingOptions = {}): (() => void) => {
  const timerId = useRef<NodeJS.Timeout>();
  const countId = useRef(0);

  // Normalize polling function to be an async function.
  const asyncPollingFn = useMemo(() => {
    if (isAsyncFunction(pollingFn)) return pollingFn;
    return async () => await pollingFn();
  }, [ pollingFn ]);

  const pollingRoutine = useCallback(async (): Promise<void> => {
    countId.current++;
    await asyncPollingFn();
    timerId.current = setTimeout(() => {
      pollingRoutine();
    }, delay || DEFAULT_DELAY);
  }, [ asyncPollingFn, delay ]);

  const stopPolling = useCallback((): void => {
    if (timerId.current) {
      clearTimeout(timerId.current);
      timerId.current = undefined;
    }
  }, []);

  useEffect(() => {
    stopPolling();
    pollingRoutine();
    return stopPolling;
  }, [ delay, pollingRoutine, stopPolling ]);

  return stopPolling;
};

export default usePolling;
