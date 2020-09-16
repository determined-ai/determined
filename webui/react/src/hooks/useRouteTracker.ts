import { useEffect } from 'react';
import { useHistory } from 'react-router-dom';

import { recordPageAccess } from 'Analytics';

const useRouteTracker = (): void => {
  const { listen, location } = useHistory();

  useEffect(() => {
    // The very first page access which doesn't trigger location change.
    recordPageAccess(location.pathname);

    // Listen for route changes
    const unlisten = listen((newLocation) => recordPageAccess(newLocation.pathname));

    // Return listener remover
    return unlisten;

    /*
     * Explicitly avoid adding `location.pathname` as a dependency to avoid
     * having the listener recreated each time `useRouteTracker` gets called
     * during a render.
     */
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, [ listen ]);
};

export default useRouteTracker;
