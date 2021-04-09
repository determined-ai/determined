import React, { useEffect, useState } from 'react';
import { useHistory, useLocation } from 'react-router-dom';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { paths } from 'routes/utils';
import { logout } from 'services/api';
import { updateDetApi } from 'services/apiConfig';

const SignOut: React.FC = () => {
  const history = useHistory();
  const location = useLocation();
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
      history.push(paths.login(), location.state);
    };

    if (!isSigningOut) signOut();

  }, [ history, location.state, isSigningOut, storeDispatch ]);

  return null;
};

export default SignOut;
