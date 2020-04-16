import { useCallback, useEffect, useRef } from 'react';

import { isAsyncFunction } from 'utils/data';

const DEFAULT_DELAY = 5000;

type PollingFn = (() => Promise<void>) | (() => void);

interface PollingOptions {
  delay?: number;
  triggers?: unknown[];
}

const usePolling =
  (pollingFn: PollingFn, { delay, triggers }: PollingOptions = {}): (() => void) => {
    const timerId = useRef<NodeJS.Timeout>();
    const countId = useRef(0);

    const pollingRoutine = useCallback(async (): Promise<void> => {
      countId.current++;
      isAsyncFunction(pollingFn) ? await pollingFn() : pollingFn();
      timerId.current = setTimeout(() => {
        pollingRoutine();
      }, delay || DEFAULT_DELAY);
    }, [ pollingFn, delay ]);

    const stopPolling = (): void => {
      if (timerId.current) {
        clearTimeout(timerId.current);
        timerId.current = undefined;
      }
    };

    useEffect(() => {
      stopPolling();
      pollingRoutine();
      return stopPolling;
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
    }, [ pollingRoutine, ...(triggers || []) ]);

    return stopPolling;
  };

export default usePolling;
