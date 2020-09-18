import { useCallback, useEffect, useRef } from 'react';

const DEFAULT_DELAY = 5000;

type PollingFn = (() => Promise<void>) | (() => void);

interface PollingOptions {
  delay?: number;
}

const usePolling = (pollingFn: PollingFn, { delay }: PollingOptions = {}): (() => void) => {
  const timerId = useRef<NodeJS.Timeout>();

  const stopPolling = useCallback((): void => {
    if (timerId.current) {
      clearTimeout(timerId.current);
      timerId.current = undefined;
    }
  }, []);

  const pollingRoutine = useCallback(async (): Promise<void> => {
    await pollingFn();

    if (timerId.current) clearTimeout(timerId.current);

    timerId.current = setTimeout(() => pollingRoutine(), delay || DEFAULT_DELAY);
  }, [ pollingFn, delay ]);

  useEffect(() => {
    pollingRoutine();
    return stopPolling;
  }, [ pollingRoutine, stopPolling ]);

  return stopPolling;
};

export default usePolling;
