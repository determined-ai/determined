import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

import history from 'shared/routes/history';

import useTelemetry from './useTelemetry';

const useRouteTracker = (): void => {
  const location = useLocation();
  const { trackPage } = useTelemetry();

  useEffect(() => {
    // Listen for route changes.
    const unlisten = history.listen(() => trackPage());

    // Clean up listener during unmount.
    return () => unlisten();
  }, [location.pathname, location.search, trackPage]);
};

export default useRouteTracker;
