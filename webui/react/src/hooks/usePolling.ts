import { useEffect, useRef } from 'react';

import { isAsyncFunction } from 'utils/data';

const DEFAULT_DELAY = 5000;

type PollingFn = (() => Promise<void>) | (() => void);

interface PollingOptions {
  delay?: number;
  triggers?: unknown[];
}

const usePolling = (pollingFn: PollingFn, options: PollingOptions = {}): (() => void) => {
  const timerId = useRef<number>();
  const countId = useRef(0);

  const pollingRoutine = async (): Promise<void> => {
    countId.current++;
    isAsyncFunction(pollingFn) ? await pollingFn() : pollingFn();
    /* eslint-disable-next-line @typescript-eslint/no-use-before-define */
    startPolling();
  };

  const startPolling = (): void => {
    timerId.current = setTimeout(() => {
      pollingRoutine();
    }, options.delay || DEFAULT_DELAY);
  };

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
  }, options.triggers || []);

  return stopPolling;
};

export default usePolling;
