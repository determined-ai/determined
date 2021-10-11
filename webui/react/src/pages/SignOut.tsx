import React, { useEffect, useState } from 'react';
import { useHistory, useLocation } from 'react-router-dom';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { paths, routeAll } from 'routes/utils';
import { logout } from 'services/api';
import { updateDetApi } from 'services/apiConfig';

const SignOut: React.FC = () => {
  const history = useHistory();
  const location = useLocation();
  const { info } = useStore();
  const storeDispatch = useStoreDispatch();
  const [ isSigningOut, setIsSigningOut ] = useState(false);

  useEffect(() => {
    const signOut = async (): Promise<void> => {
      setIsSigningOut(true);
      try {
        await logout({});
      } catch (e) {
        handleError({
          error: e,
          isUserTriggered: false,
          level: ErrorLevel.Warn,
          message: e.message,
          silent: true,
          type: ErrorType.Server,
        });
      }
      updateDetApi({ apiKey: undefined });
      storeDispatch({ type: StoreAction.ResetAuth });

      if (info.externalLogoutUri) {
        routeAll(info.externalLogoutUri);
      } else {
        history.push(paths.login(), location.state);
      }
    };

    if (!isSigningOut) signOut();

  }, [ history, info.externalLogoutUri, location.state, isSigningOut, storeDispatch ]);

  return null;
};

export default SignOut;
