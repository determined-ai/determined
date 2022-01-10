import { useEffect } from 'react';
import { useHistory } from 'react-router-dom';

import Telemetry from 'classes/Telemetry';

const useRouteTracker = (): void => {
  const { listen } = useHistory();

  useEffect(() => {
    // Listen for route changes.
    const unlisten = listen(() => Telemetry.page());

    // Clean up listener during unmount.
    return () => unlisten();
  }, [ listen ]);
};

export default useRouteTracker;
