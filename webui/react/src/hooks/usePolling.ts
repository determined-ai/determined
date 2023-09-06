import { useCallback, useEffect, useRef } from 'react';

import useUI from 'components/kit/contexts/UI';

type PollingFn = (() => Promise<void>) | (() => void);

interface PollingHooks {
  isPolling: boolean;
  startPolling: () => void;
  stopPolling: (options?: StopOptions) => void;
}

interface PollingOptions {
  /** whether to continue polling when the page/tab is out of focus. */
  continueWhenHidden?: boolean;
  interval?: number;
  rerunOnNewFn?: boolean;
  runImmediately?: boolean;
}

/*
 * When calling `stopPolling` with `terminateGracefully` set to true,
 * the polling will be marked as stopped but we avoid killing the timer.
 * This means that the polling function will allowed to run one last time
 * before terminating.
 */
interface StopOptions {
  terminateGracefully?: boolean;
}

const DEFAULT_OPTIONS: PollingOptions = {
  continueWhenHidden: false,
  interval: 5000,
  rerunOnNewFn: false,
  runImmediately: true,
};

/**
 * Polling hook that polls a given polling function.
 * @param pollingFn
 *    The function to poll. Pass an async/await function to ensure
 *    the previous poll resolves first before making a new call.
 */
const usePolling = (pollingFn: PollingFn, options: PollingOptions = {}): PollingHooks => {
  const savedPollingFn = useRef<PollingFn>(pollingFn);
  const pollingOptions = useRef<PollingOptions>({ ...DEFAULT_OPTIONS, ...options });
  const timer = useRef<NodeJS.Timeout>();
  const isPolling = useRef(false);
  const isPollingBeforeHidden = useRef(false);
  const pollingFnIndicator = pollingOptions.current.rerunOnNewFn ? pollingFn : undefined;
  const { ui } = useUI();

  const clearTimer = useCallback(() => {
    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = undefined;
    }
  }, []);

  const poll = useCallback(() => {
    clearTimer();

    timer.current = setTimeout(async () => {
      await savedPollingFn.current();
      timer.current = undefined;
      if (isPolling.current) poll();
    }, pollingOptions.current.interval) as unknown as NodeJS.Timeout;
  }, [clearTimer]);

  const startPolling = useCallback(async () => {
    isPolling.current = true;
    if (pollingOptions.current.runImmediately) await savedPollingFn.current();
    poll();
  }, [poll]);

  const stopPolling = useCallback(
    (options: StopOptions = {}) => {
      isPolling.current = false;
      if (!options.terminateGracefully) clearTimer();
    },
    [clearTimer],
  );

  // Update polling function if a new one is passed in.
  useEffect(() => {
    savedPollingFn.current = pollingFn;
  }, [pollingFn]);

  // Start polling when mounted and stop polling when umounted.
  useEffect(() => {
    startPolling();
    return () => stopPolling();
  }, [startPolling, stopPolling, pollingFnIndicator]);

  // control polling when the page is hidden
  useEffect(() => {
    if (pollingOptions.current.continueWhenHidden) return;
    if (ui.isPageHidden) {
      // Save the state of whether polling was active before page is hidden.
      isPollingBeforeHidden.current = isPolling.current;

      // Stop polling if currently active.
      if (isPolling.current) stopPolling();
    } else {
      /**
       * Start polling again if everything below is true.
       *  - the page is visible
       *  - currently not polling
       *  - was polling previously before page became hidden
       */
      if (!isPolling.current && isPollingBeforeHidden.current) startPolling();
    }
  }, [startPolling, stopPolling, ui.isPageHidden]);

  return { isPolling: isPolling.current, startPolling, stopPolling };
};

export default usePolling;
