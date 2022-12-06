import React, { useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

import { useStore } from 'contexts/Store';
import { paths, routeAll } from 'routes/utils';
import { logout } from 'services/api';
import { updateDetApi } from 'services/apiConfig';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { isAuthFailure } from 'shared/utils/service';
import { useAuth } from 'stores/auth';
import handleError from 'utils/error';

const SignOut: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { info } = useStore();
  const { resetAuth } = useAuth();
  const [isSigningOut, setIsSigningOut] = useState(false);

  useEffect(() => {
    const signOut = async (): Promise<void> => {
      setIsSigningOut(true);
      try {
        await logout({});
      } catch (e) {
        if (!isAuthFailure(e)) {
          handleError(e, {
            isUserTriggered: false,
            level: ErrorLevel.Warn,
            silent: true,
            type: ErrorType.Server,
          });
        }
      }
      updateDetApi({ apiKey: undefined });
      resetAuth();

      if (info.externalLogoutUri) {
        routeAll(info.externalLogoutUri);
      } else {
        navigate(paths.login(), { state: location.state });
      }
    };

    if (!isSigningOut) signOut();
  }, [navigate, info.externalLogoutUri, location.state, isSigningOut, resetAuth]);

  return null;
};

export default SignOut;
