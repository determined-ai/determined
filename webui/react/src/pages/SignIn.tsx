import { notification } from 'antd';
import queryString from 'query-string';
import React, { useCallback, useEffect } from 'react';

import AuthToken from 'components/AuthToken';
import DeterminedAuth from 'components/DeterminedAuth';
import Logo, { LogoTypes } from 'components/Logo';
import Auth from 'contexts/Auth';
import FullPageSpinner from 'contexts/FullPageSpinner';
import usePolling from 'hooks/usePolling';
import { routeAll } from 'routes';
import { getCookie } from 'utils/browser';

import css from './SignIn.module.scss';

interface Queries {
  cli?: boolean;
  redirect?: string;
}

const DEFAULT_REDIRECT = '/det/dashboard';

const SignIn: React.FC = () => {
  const auth = Auth.useStateContext();
  const setShowSpinner = FullPageSpinner.useActionContext();
  const queries: Queries = queryString.parse(location.search);

  // Redirect the user to the app if auth cookie already exists.
  const checkAuth = useCallback(() => {
    if (getCookie('auth')) routeAll(queries.redirect || DEFAULT_REDIRECT);
  }, [ queries.redirect ]);

  /*
   * Check every so often to see if the user cookie exists.
   * For example, the user can authenticate in a different session,
   * and this will pick up that auth and automatically redirect them into
   * their previous app.
   */
  // const task = useAsyncTask(checkAuth);
  const stopPolling = usePolling(checkAuth);

  /*
   * Check for when `isAuthenticated` becomes true and redirect
   * the user to the most recent requested page.
   */
  useEffect(() => {
    if (!auth.isAuthenticated) return;

    // Stop the spinner, prepping for user redirect.
    setShowSpinner({ type: FullPageSpinner.ActionType.Hide });

    // Show auth token via notification if requested via query parameters.
    if (queries.cli) notification.open({ description: <AuthToken />, duration: 0, message: '' });

    // Reroute the authenticated user to the app.
    routeAll(queries.redirect || DEFAULT_REDIRECT);

    return stopPolling;
  }, [ auth.isAuthenticated, queries, setShowSpinner, stopPolling ]);

  return (
    <div className={css.base}>
      <div className={css.content}>
        <Logo type={LogoTypes.OnLightVertical} />
        <DeterminedAuth />
      </div>
    </div>
  );
};

export default SignIn;
