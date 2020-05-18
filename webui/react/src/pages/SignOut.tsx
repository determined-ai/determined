import React, { useEffect, useState } from 'react';
import { useHistory, useLocation } from 'react-router-dom';

import Spinner from 'components/Spinner';
import Auth from 'contexts/Auth';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { logout } from 'services/api';

const SignOut: React.FC = () => {
  const history = useHistory();
  const location = useLocation();
  const auth = Auth.useStateContext();
  const setAuth = Auth.useActionContext();
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
      setAuth({ type: Auth.ActionType.Reset });
      history.push(`/det/login${location.search}`);
    };
    if (!isSigningOut) signOut();
  }, [ auth.isAuthenticated, history, isSigningOut, location, setAuth ]);

  return <Spinner fullPage />;
};

export default SignOut;
