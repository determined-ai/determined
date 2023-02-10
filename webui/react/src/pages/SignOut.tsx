import React, { useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

import { paths, routeAll } from 'routes/utils';
import { logout } from 'services/api';
import { updateDetApi } from 'services/apiConfig';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { isAuthFailure } from 'shared/utils/service';
import { useAuth } from 'stores/auth';
import { initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import { PermissionsStore } from 'stores/permissions';
import { useUpdateCurrentUser } from 'stores/users';
import { useResetWorkspaces } from 'stores/workspaces';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

const SignOut: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { resetAuth } = useAuth();
  const info = Loadable.getOrElse(initInfo, useDeterminedInfo());
  const [isSigningOut, setIsSigningOut] = useState(false);
  const updateCurrentUser = useUpdateCurrentUser();
  const resetWorkspaces = useResetWorkspaces();

  useEffect(() => {
    const signOut = async (): Promise<void> => {
      setIsSigningOut(true);
      PermissionsStore.resetMyAssignmentsAndRoles();
      updateCurrentUser(null);
      resetWorkspaces();
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
        navigate(paths.login() + '?r=' + Math.random(), { state: location.state });
      }
    };

    if (!isSigningOut) signOut();
  }, [
    navigate,
    info.externalLogoutUri,
    location.state,
    updateCurrentUser,
    isSigningOut,
    resetWorkspaces,
    resetAuth,
  ]);

  return null;
};

export default SignOut;
