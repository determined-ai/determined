import { useCallback, useEffect, useMemo } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';

interface DocumentHidden {
  hidden?: unknown;
  msHidden?: unknown;
  webkitHidden?: unknown;
}

const usePageVisibility = (): void => {
  const storeDispatch = useStoreDispatch();

  const [ hidden, visibilityChange ] = useMemo(() => {
    if (typeof (document as DocumentHidden).hidden !== 'undefined') {
      return [ 'hidden', 'visibilitychange' ];
    } else if (typeof (document as DocumentHidden).msHidden !== 'undefined') {
      return [ 'msHidden', 'msvisibilitychange' ];
    } else if (typeof (document as DocumentHidden).webkitHidden !== 'undefined') {
      return [ 'webkitHidden', 'webkitvisibilitychange' ];
    }
    return [ undefined, undefined ];
  }, []);

  const handleVisibilityChange = useCallback(() => {
    if (!hidden) return;
    storeDispatch({
      type: StoreAction.SetPageVisibility,
      value: !!(document as DocumentHidden)[hidden as keyof DocumentHidden],
    });
  }, [ hidden, storeDispatch ]);

  useEffect(() => {
    if (visibilityChange) {
      document.addEventListener(visibilityChange, handleVisibilityChange);
    }

    return () => {
      if (visibilityChange) {
        document.removeEventListener(visibilityChange, handleVisibilityChange);
      }
    };
  }, [ handleVisibilityChange, visibilityChange ]);
};

export default usePageVisibility;
