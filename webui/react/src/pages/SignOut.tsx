import { useObservable } from 'micro-observables';
import React, { useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

import useAuthCheck from 'hooks/useAuthCheck';
import { paths, routeAll } from 'routes/utils';
import { logout } from 'services/api';
import { updateDetApi } from 'services/apiConfig';
import authStore from 'stores/auth';
import determinedStore from 'stores/determinedInfo';
import permissionStore from 'stores/permissions';
import roleStore from 'stores/roles';
import userStore from 'stores/users';
import userSettings from 'stores/userSettings';
import workspaceStore from 'stores/workspaces';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { isAuthFailure } from 'utils/service';

const SignOut: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const info = useObservable(determinedStore.info);
  const [isSigningOut, setIsSigningOut] = useState(false);
  const queries = useMemo(() => new URLSearchParams(location.search), [location.search]);
  const checkAuth = useAuthCheck();

  useEffect(() => {
    const signOut = async (): Promise<void> => {
      setIsSigningOut(true);
      roleStore.reset();
      permissionStore.reset();
      userStore.reset();
      workspaceStore.reset();
      userSettings.reset();
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
      authStore.reset();

      if (info.externalLogoutUri) {
        const isAuthenticated = await checkAuth();
        if (isAuthenticated) routeAll(info.externalLogoutUri);
      } else {
        const searchParameters = [`?r=${Math.random()}`];
        if (queries.has('remote_expired')) {
          searchParameters.push(`&remote_expired=${queries.get('remote_expired') || ''}`);
        }
        if (queries.has('redirect'))
          searchParameters.push(`&redirect=${queries.get('redirect') || ''}`);
        navigate(paths.login() + searchParameters.join(''), { state: location.state });
      }
    };

    if (!isSigningOut) signOut();
  }, [checkAuth, navigate, info.externalLogoutUri, location.state, isSigningOut, queries]);

  return null;
};

export default SignOut;
