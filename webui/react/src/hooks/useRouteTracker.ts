import { useEffect, useRef } from 'react';
import { useHistory } from 'react-router-dom';

import { recordPageAccess } from 'Analytics';

const useRouteTracker = (): void => {
  const { listen, location } = useHistory();
  const pathnameRef = useRef(location.pathname);

  useEffect(() => {
    // The very first page access which doesn't trigger location change.
    recordPageAccess(pathnameRef.current);

    // Listen for route changes.
    const unlisten = listen((newLocation) => recordPageAccess(newLocation.pathname));

    // Clean up listener during unmount.
    return () => unlisten();
  }, [ listen ]);

  // Update pathname reference when location changes.
  useEffect(() => {
    pathnameRef.current = location.pathname;
  }, [ location.pathname ]);
};

export default useRouteTracker;
