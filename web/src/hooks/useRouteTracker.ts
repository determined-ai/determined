import { useEffect } from 'react';
import { useHistory } from 'react-router-dom';

import useTelemetry from './useTelemetry';

const useRouteTracker = (): void => {
  const { listen } = useHistory();
  const { trackPage } = useTelemetry();

  useEffect(() => {
    // Listen for route changes.
    const unlisten = listen(() => trackPage());

    // Clean up listener during unmount.
    return () => unlisten();
  }, [ listen, trackPage ]);
};

export default useRouteTracker;
