import React, { useEffect, useState } from 'react';
import { useHistory, useLocation } from 'react-router-dom';

import Auth from 'contexts/Auth';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { paths } from 'routes/utils';
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
      history.push(paths.login(), location.state);
    };

    if (!isSigningOut) signOut();

  }, [ auth.isAuthenticated, history, location.state, isSigningOut, setAuth ]);

  return null;
};

export default SignOut;
