import { useCallback, useEffect, useRef } from 'react';

import { isAsyncFunction } from 'utils/data';

const DEFAULT_DELAY = 5000;

type PollingFn = (() => Promise<void>) | (() => void);

interface PollingOptions {
  delay?: number;
}

const usePolling = (pollingFn: PollingFn, { delay }: PollingOptions = {}): (() => void) => {
  const timerId = useRef<NodeJS.Timeout>();
  const countId = useRef(0);

  const stopPolling = useCallback((): void => {
    if (timerId.current) {
      clearTimeout(timerId.current);
      timerId.current = undefined;
    }
  }, []);

  const pollingRoutine = useCallback(async (): Promise<void> => {
    countId.current++;

    const count = countId.current;

    isAsyncFunction(pollingFn) ? await pollingFn() : pollingFn();

    timerId.current = setTimeout(() => {
      /*
         * When the polling function changes rapidly it's possible for several timers
         * to be active. The count checks ensures that only the most recently set
         * timer is allowed to continue polling behavior.
         */
      if (count === countId.current) pollingRoutine();
    }, delay || DEFAULT_DELAY);
  }, [ pollingFn, delay ]);

  useEffect(() => {
    pollingRoutine();
    return stopPolling;
  }, [ pollingRoutine, stopPolling ]);

  return stopPolling;
};

export default usePolling;
