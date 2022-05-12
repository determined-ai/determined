import { Button, notification } from 'antd';
import queryString from 'query-string';
import React, { useEffect, useMemo, useState } from 'react';
import { useLocation } from 'react-router-dom';

import AuthToken from 'components/AuthToken';
import DeterminedAuth from 'components/DeterminedAuth';
import Logo, { LogoType } from 'components/Logo';
import Page from 'components/Page';
import PageMessage from 'components/PageMessage';
import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { handleRelayState, samlUrl } from 'ee/SamlAuth';
import useAuthCheck from 'hooks/useAuthCheck';
import usePolling from 'hooks/usePolling';
import { defaultRoute } from 'routes';
import LogoGoogle from 'shared/assets/images/logo-sso-google-white.svg';
import LogoOkta from 'shared/assets/images/logo-sso-okta-white.svg';
import { getPath } from 'shared/utils/data';
import { capitalize } from 'shared/utils/string';

import { routeAll } from '../routes/utils';
import { RecordKey } from '../shared/types';
import { locationToPath, routeToReactUrl } from '../shared/utils/routes';

import css from './SignIn.module.scss';

interface Queries {
  cli?: boolean;
  jwt?: string;
  redirect?: string;
}

const logoConfig: Record<RecordKey, string> = {
  google: LogoGoogle,
  okta: LogoOkta,
};

const SignIn: React.FC = () => {
  const location = useLocation<{ loginRedirect: Location }>();
  const { auth, info } = useStore();
  const storeDispatch = useStoreDispatch();
  const [ canceler ] = useState(new AbortController());

  const queries: Queries = queryString.parse(location.search);
  const ssoQueries = handleRelayState(queries) as Record<string, boolean | string | undefined>;
  const ssoQueryString = queryString.stringify(ssoQueries);

  const externalAuthError = useMemo(() => {
    return auth.checked && !auth.isAuthenticated && !info.externalLoginUri && queries.jwt;
  }, [ auth.checked, auth.isAuthenticated, info.externalLoginUri, queries.jwt ]);

  /*
   * Check every so often to see if the user is authenticated.
   * For example, the user can authenticate in a different session,info
   * and this will pick up that auth and automatically redirect them into
   * their previous app. We don't run immediately because the router also
   * performs an auth check there as well upon the first page load.
   */
  usePolling(useAuthCheck(canceler), { interval: 1000, runImmediately: false });

  /*
   * Check for when `isAuthenticated` becomes true and redirect
   * the user to the most recent requested page.
   */
  useEffect(() => {
    if (auth.isAuthenticated) {
      // Stop the spinner, prepping for user redirect.
      storeDispatch({ type: StoreAction.HideUISpinner });

      // Show auth token via notification if requested via query parameters.
      if (queries.cli) notification.open({ description: <AuthToken />, duration: 0, message: '' });

      // Reroute the authenticated user to the app.
      const loginRedirect = getPath<Location>(location, 'state.loginRedirect');
      if (!queries.redirect) {
        routeToReactUrl(locationToPath(loginRedirect) || defaultRoute.path);
      } else {
        routeAll(queries.redirect);
      }
    } else if (auth.checked) {
      storeDispatch({ type: StoreAction.HideUISpinner });
    }
  }, [ auth, info, location, queries, storeDispatch ]);

  useEffect(() => {
    storeDispatch({ type: StoreAction.HideUIChrome });
    return () => storeDispatch({ type: StoreAction.ShowUIChrome });
  }, [ storeDispatch ]);

  // Stop the polling upon a dismount of this page.
  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  /*
   * Don't render sign in page if...
   *   1. jwt query param detected
   *   2. cluster has `externalLoginUri` defined
   *   3. authentication hasn't occurred yet
   * This will prevent the form from showing for a split second when
   * accessing a page from the browser when the user is already verified.
   */
  if (queries.jwt || info.externalLoginUri || !auth.checked) return null;

  /*
   * An external auth error occurs when there are external auth urls,
   * auth fails with a jwt.
   */
  if (externalAuthError) return (
    <PageMessage title="Cluster Not Available">
      <p>Cluster is not ready. Please try again later.</p>
    </PageMessage>
  );

  return (
    <Page docTitle="Sign In">
      <div className={css.base}>
        <div className={css.content}>
          <Logo branding={info.branding} type={LogoType.OnLightVertical} />
          <DeterminedAuth canceler={canceler} />
          {info.ssoProviders?.map(ssoProvider => {
            const key = ssoProvider.name.toLowerCase();
            const logo = logoConfig[key] ? <img alt={key} src={logoConfig[key]} /> : '';
            return (
              <Button
                className={css.ssoButton}
                href={samlUrl(ssoProvider.ssoUrl, ssoQueryString)}
                key={key}
                size="large"
                type="primary">
                Sign in with {logo} {capitalize(key)}
              </Button>
            );
          })}
        </div>
      </div>
    </Page>
  );
};

export default SignIn;
