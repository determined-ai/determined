import { useSyncExternalStore } from 'react';

import useResize from './useResize';

const MOBILE_BREAKPOINT = 480;

const useMobile = (): boolean => {
  const { width } = useResize();

  return width < MOBILE_BREAKPOINT;
};

export const useMediaQuery = (mq: string): boolean => {
  const matchMediaQuery = useSyncExternalStore(subscribe(mq), getSnapshot(mq));

  return matchMediaQuery;
};

function getSnapshot(mq: string) {
  return () => window.matchMedia?.(mq).matches;
}

function subscribe(mq: string) {
  return (callback: () => void) => {
    window.matchMedia?.(mq).addEventListener('change', callback);

    return () => {
      window.matchMedia?.(mq).removeEventListener('change', callback);
    };
  };
}

export default useMobile;
