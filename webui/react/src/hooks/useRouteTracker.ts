import { useEffect } from 'react';
import { useHistory } from 'react-router-dom';

import Info from 'contexts/Info';

const recordPageAccess = (pathname: string): void => {
  // Check to make sure Segment `analytics` is available
  if (!window.analytics) return;

  // Record page access
  window.analytics.page(pathname);
};

const useRouteTracker = (): void => {
  const { listen, location } = useHistory();
  const info = Info.useStateContext();

  useEffect(() => {
    // The very first page access which doesn't trigger location change.
    if (info.telemetry.enabled) recordPageAccess(location.pathname);

    // Listen for route changes
    const unlisten = listen((location) => {
      if (info.telemetry.enabled) recordPageAccess(location.pathname);
    });

    // Return listener remover
    return unlisten;
  }, [ listen, info.telemetry.enabled, location.pathname ]);
};

export default useRouteTracker;
