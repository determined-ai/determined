import { useCallback, useEffect, useRef } from 'react';

const DEFAULT_DELAY = 5000;

type PollingFn = (() => Promise<void>) | (() => void);

interface PollingOptions {
  delay?: number;
}

const usePolling = (pollingFn: PollingFn, { delay }: PollingOptions = {}): (() => void) => {
  const func = useRef<PollingFn>(pollingFn);
  const timer = useRef<NodeJS.Timeout>();
  const active = useRef(true);

  const stopPolling = useCallback(() => {
    active.current = false;

    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = undefined;
    }
  }, []);

  const runPolling = useCallback(async (): Promise<void> => {
    await func.current();

    if (active.current) {
      timer.current = setTimeout(() => runPolling(), delay || DEFAULT_DELAY);
    }
  }, [ delay, func ]);

  useEffect(() => {
    runPolling();
    return () => stopPolling();
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, []);

  return stopPolling;
};

export default usePolling;
