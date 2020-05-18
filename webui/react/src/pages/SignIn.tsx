import { notification } from 'antd';
import axios from 'axios';
import queryString from 'query-string';
import React, { useCallback, useEffect, useState } from 'react';

import AuthToken from 'components/AuthToken';
import DeterminedAuth from 'components/DeterminedAuth';
import Logo, { LogoTypes } from 'components/Logo';
import Auth from 'contexts/Auth';
import ShowSpinner from 'contexts/ShowSpinner';
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
  const setShowSpinner = ShowSpinner.useActionContext();
  const [ isLoading, setIsLoading ] = useState(false);
  const [ hasCheckedAuth, setHasCheckedAuth ] = useState(false);
  const queries: Queries = queryString.parse(location.search);

  const onLoadingChange = useCallback((loading: boolean): void => {
    setIsLoading(loading);
  }, [ setIsLoading ]);

  /*
   * Map the spinner show state to the local `isLoading` state.
   */
  useEffect(() => {
    setShowSpinner({ type: isLoading ? ShowSpinner.ActionType.Show : ShowSpinner.ActionType.Hide });
  }, [ isLoading, setShowSpinner ]);

  /*
   * Verify existing user authentication via cookies and update
   * authentication state with the verified user.
   */
  useEffect(() => {
    if (hasCheckedAuth) return;

    const source = axios.CancelToken.source();
    const checkAuth = async (): Promise<void> => {
      try {
        const user = await getCurrentUser({ cancelToken: source.token });
        setAuth({ type: Auth.ActionType.Set, value: { isAuthenticated: true, user } });
      } finally {
        setIsLoading(false);
        setHasCheckedAuth(true);
      }
    };

    checkAuth();

    return (): void => source.cancel();
  }, [ hasCheckedAuth, setAuth ]);

  /*
   * Check for when `isAuthenticated` becomes true and redirect
   * the user to the most recent requested page.
   */
  useEffect(() => {
    if (!auth.isAuthenticated) return;

    // Stop the spinner, prepping for user redirect.
    setShowSpinner({ type: ShowSpinner.ActionType.Hide });

    // Show auth token via notification if requested via query parameters.
    if (queries.cli) notification.open({ description: <AuthToken />, duration: 0, message: '' });

    // Reroute the authenticated user to the app.
    routeAll(queries.redirect || DEFAULT_REDIRECT);
  }, [ auth.isAuthenticated, queries, setShowSpinner ]);

  return (
    <div className={css.base}>
      <div className={css.content}>
        <Logo type={LogoTypes.OnLightVertical} />
        <DeterminedAuth onLoadingChange={onLoadingChange} />
      </div>
    </div>
  );
};

export default SignIn;
