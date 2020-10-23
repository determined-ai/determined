import { notification } from 'antd';
import queryString from 'query-string';
import React, { useEffect } from 'react';
import { Helmet } from 'react-helmet';
import { useLocation } from 'react-router-dom';

import AuthToken from 'components/AuthToken';
import DeterminedAuth from 'components/DeterminedAuth';
import Logo, { LogoTypes } from 'components/Logo';
import Auth from 'contexts/Auth';
import UI from 'contexts/UI';
import useAuthCheck from 'hooks/useAuthCheck';
import usePolling from 'hooks/usePolling';
import { defaultRoutePath } from 'routes';
import { locationToPath, routeAll } from 'routes/utils';
import { getPath } from 'utils/data';

import css from './SignIn.module.scss';

interface Queries {
  cli?: boolean;
  redirect?: string;
}

const SignIn: React.FC = () => {
  const location = useLocation<{ loginRedirect: Location }>();
  const auth = Auth.useStateContext();
  const ui = UI.useStateContext();
  const setUI = UI.useActionContext();
  const queries: Queries = queryString.parse(location.search);

  /*
   * Check every so often to see if the user is authenticated.
   * For example, the user can authenticate in a different session,
   * and this will pick up that auth and automatically redirect them into
   * their previous app.
   */
  const checkAuth = useAuthCheck();
  const stopPolling = usePolling(checkAuth, { delay: 1000 });

  /*
   * Check for when `isAuthenticated` becomes true and redirect
   * the user to the most recent requested page.
   */
  useEffect(() => {
    if (auth.isAuthenticated) {
      // Stop the spinner, prepping for user redirect.
      setUI({ type: UI.ActionType.HideSpinner });

      // Show auth token via notification if requested via query parameters.
      if (queries.cli) notification.open({ description: <AuthToken />, duration: 0, message: '' });

      // Reroute the authenticated user to the app.
      const loginRedirect = getPath<Location>(location, 'state.loginRedirect');
      const redirect = queries.redirect || locationToPath(loginRedirect);
      routeAll(redirect || defaultRoutePath);
    } else if (auth.checked) {
      setUI({ type: UI.ActionType.HideSpinner });
    }
  }, [
    auth.checked,
    auth.isAuthenticated,
    location,
    queries,
    setUI,
    ui,
  ]);

  // Stop the polling upon a dismount of this page.
  useEffect(() => stopPolling, [ stopPolling ]);

  /*
   * Before showing the sign in form, make sure one auth check is done.
   * This will prevent the form from showing for a split second when
   * accessing a page from the browser when the user is already verified.
   */
  return auth.checked ?
    <div className={css.base}>
      <Helmet>
        <title>Sign In - Determined</title>
      </Helmet>
      <div className={css.content}>
        <Logo type={LogoTypes.OnLightVertical} />
        <DeterminedAuth />
      </div>
    </div> : null;
};

export default SignIn;
