import { useCallback, useEffect, useRef } from 'react';

const DEFAULT_DELAY = 5000;

type PollingFn = (() => Promise<void>) | (() => void);
type StopFn = () => void;

interface PollingOptions {
  delay?: number;
}

const usePolling = (pollingFn: PollingFn, { delay }: PollingOptions = {}): StopFn => {
  const savedPollingFn = useRef<PollingFn>(pollingFn);
  const timer = useRef<NodeJS.Timeout>();
  const active = useRef(false);

  const clearTimer = useCallback(() => {
    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = undefined;
    }
  }, []);

  const poll = useCallback(async (): Promise<void> => {
    await savedPollingFn.current();

    if (active.current) {
      timer.current = setTimeout(() => {
        timer.current = undefined;
        poll();
      }, delay || DEFAULT_DELAY);
    }
  }, [ delay ]);

  const startPolling = useCallback(() => {
    clearTimer();
    active.current = true;
    poll();
  }, [ clearTimer, poll ]);

  const stopPolling = useCallback(() => {
    active.current = false;
    clearTimer();
  }, [ clearTimer ]);

  // Update polling function if a new one is passed in.
  useEffect(() => {
    savedPollingFn.current = pollingFn;
    if (!active.current) startPolling();
  }, [ pollingFn, startPolling ]);

  // Start polling when mounted and stop polling when umounted.
  useEffect(() => {
    startPolling();
    return () => stopPolling();
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, []);

  return stopPolling;
};

export default usePolling;
