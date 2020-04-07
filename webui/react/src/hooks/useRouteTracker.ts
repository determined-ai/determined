import { useEffect } from 'react';
import { useHistory } from 'react-router-dom';

const recordPageAccess = (pathname: string): void => {
  // Check to make sure Segment `analytics` is available
  if (!window.analytics) return;

  // Record page access
  window.analytics.page(pathname);
};

const useRouteTracker = (): void => {
  const { listen, location } = useHistory();

  useEffect(() => {
    // The very first page access which doesn't trigger location change.
    recordPageAccess(location.pathname);

    // Listen for route changes
    const unlisten = listen((location) => {
      recordPageAccess(location.pathname);
    });

    // Return listener remover
    return unlisten;
  }, [ listen ]);
};

export default useRouteTracker;
