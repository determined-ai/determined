import { useCallback, useEffect, useRef } from 'react';

const DEFAULT_INTERVAL = 5000;

type PollingFn = (() => Promise<void>) | (() => void);

interface PollingHooks {
  isPolling: boolean;
  startPolling: () => void;
  stopPolling: () => void;
}

interface PollingOptions {
  interval?: number;
  runImmediately?: boolean;
  stopPreviousPoll?: boolean;
}

const usePolling = (pollingFn: PollingFn, {
  interval = DEFAULT_INTERVAL,
  runImmediately = true,
  stopPreviousPoll = false,
}: PollingOptions = {}): PollingHooks => {
  const savedPollingFn = useRef<PollingFn>(pollingFn);
  const timer = useRef<NodeJS.Timeout>();
  const isPolling = useRef(false);

  const clearTimer = useCallback(() => {
    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = undefined;
    }
  }, []);

  const poll = useCallback(async () => {
    if (stopPreviousPoll) clearTimer();
    if (runImmediately) await savedPollingFn.current();

    timer.current = setTimeout(async () => {
      await savedPollingFn.current();
      timer.current = undefined;
      if (isPolling.current) poll();
    }, interval);
  }, [ clearTimer, interval, runImmediately, stopPreviousPoll ]);

  const startPolling = useCallback(() => {
    isPolling.current = true;
    poll();
  }, [ poll ]);

  const stopPolling = useCallback(() => {
    isPolling.current = false;
    clearTimer();
  }, [ clearTimer ]);

  // Update polling function if a new one is passed in.
  useEffect(() => {
    savedPollingFn.current = pollingFn;
  }, [ pollingFn ]);

  // Start polling when mounted and stop polling when umounted.
  useEffect(() => {
    startPolling();
    return () => stopPolling();
    /*
     * The dependency array is intentionally left blank to force
     * the mount behavior and the unmount behavior.
     */
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, []);

  return { isPolling: isPolling.current, startPolling, stopPolling };
};

export default usePolling;
