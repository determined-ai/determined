import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

import useTelemetry from './useTelemetry';

const useRouteTracker = (): void => {
  const location = useLocation();
  const { trackPage } = useTelemetry();

  useEffect(() => {
    trackPage(location);
  }, [location, trackPage]);
};

export default useRouteTracker;
