import { notification } from 'antd';
import axios from 'axios';
import queryString from 'query-string';
import React, { useEffect, useState } from 'react';

import AuthToken from 'components/AuthToken';
import DeterminedAuth from 'components/DeterminedAuth';
import Logo, { LogoTypes } from 'components/Logo';
import Auth from 'contexts/Auth';
import FullPageSpinner from 'contexts/FullPageSpinner';
import handleError, { ErrorType } from 'ErrorHandler';
import { routeAll } from 'routes';
import { getCurrentUser } from 'services/api';

import css from './SignIn.module.scss';

interface Queries {
  cli?: boolean;
  redirect?: string;
}

const DEFAULT_REDIRECT = '/det/dashboard';

const SignIn: React.FC = () => {
  const auth = Auth.useStateContext();
  const setAuth = Auth.useActionContext();
  const setShowSpinner = FullPageSpinner.useActionContext();
  const [ hasCheckedAuth, setHasCheckedAuth ] = useState(false);
  const queries: Queries = queryString.parse(location.search);

  /*
   * Verify existing user authentication via cookies and update
   * authentication state with the verified user.
   */
  useEffect(() => {
    if (hasCheckedAuth) return;

    setShowSpinner({ type: FullPageSpinner.ActionType.Show });

    const source = axios.CancelToken.source();
    const checkAuth = async (): Promise<void> => {
      try {
        const user = await getCurrentUser({ cancelToken: source.token });
        setAuth({  type: Auth.ActionType.Set, value: { isAuthenticated: true, user } });
      } catch (e) {
        handleError({
          error: e,
          isUserTriggered: false,
          message: e.message,
          publicMessage: 'User is not verified.',
          publicSubject: 'Login failed',
          silent: true,
          type: ErrorType.Auth,
        });
      } finally {
        setShowSpinner({ type: FullPageSpinner.ActionType.Hide });
        setHasCheckedAuth(true);
      }
    };

    checkAuth();

    return (): void => source.cancel();
  }, [ hasCheckedAuth, setAuth, setShowSpinner ]);

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
  }, [ auth.isAuthenticated, queries, setShowSpinner ]);

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
