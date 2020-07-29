import { useEffect } from 'react';
import { useHistory } from 'react-router-dom';

import { recordPageAccess } from 'Analytics';

const useRouteTracker = (): void => {
  const { listen, location } = useHistory();

  useEffect(() => {
    // The very first page access which doesn't trigger location change.
    recordPageAccess(location.pathname);

    // Listen for route changes
    const unlisten = listen((location) => recordPageAccess(location.pathname));

    // Return listener remover
    return unlisten;
  }, [ listen, location.pathname ]);
};

export default useRouteTracker;
