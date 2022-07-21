import React, { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { useNavigate } from 'react-router-dom';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { paths, routeAll } from 'routes/utils';
import { logout } from 'services/api';
import { updateDetApi } from 'services/apiConfig';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

const SignOut: React.FC = () => {
  const navigate = useNavigate();
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
        if (!(e instanceof DetError && e.type === ErrorType.Auth)) {
          handleError(e, {
            isUserTriggered: false,
            level: ErrorLevel.Warn,
            silent: true,
            type: ErrorType.Server,
          });
        }
      }
      updateDetApi({ apiKey: undefined });
      storeDispatch({ type: StoreAction.ResetAuth });

      if (info.externalLogoutUri) {
        routeAll(info.externalLogoutUri);
      } else {
        navigate(paths.login(), { state: location.state });
      }
    };

    if (!isSigningOut) signOut();

  }, [ navigate, info.externalLogoutUri, location.state, isSigningOut, storeDispatch ]);

  return null;
};

export default SignOut;
